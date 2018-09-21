package webconsole

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	"yunion.io/x/onecloud/pkg/webconsole/command"
	o "yunion.io/x/onecloud/pkg/webconsole/options"
	"yunion.io/x/onecloud/pkg/webconsole/session"
)

const (
	ApiPathPrefix     = "/webconsole/"
	ConnectPathPrefix = "/connect/"
)

func InitHandlers(app *appsrv.Application) {
	app.AddHandler("POST", ApiPathPrefix+"k8s/<podName>/shell", auth.Authenticate(handleK8sShell))
	app.AddHandler("POST", ApiPathPrefix+"k8s/<podName>/log", auth.Authenticate(handleK8sLog))
	app.AddHandler("POST", ApiPathPrefix+"baremetal/<id>", auth.Authenticate(handleBaremetalShell))
	app.AddHandler("POST", ApiPathPrefix+"server/<id>", auth.Authenticate(handleServerShell))
}

func fetchEnv(ctx context.Context, w http.ResponseWriter, r *http.Request) (map[string]string, jsonutils.JSONObject, jsonutils.JSONObject) {
	params := appctx.AppContextParams(ctx)
	query, e := jsonutils.ParseQueryString(r.URL.RawQuery)
	if e != nil {
		log.Errorf("Parse query string %q failed: %v", r.URL.RawQuery, e)
	}
	var body jsonutils.JSONObject = nil
	if sets.NewString("PUT", "POST", "DELETE", "PATCH").Has(r.Method) {
		body, e = appsrv.FetchJSON(r)
		if e != nil {
			log.Errorf("Failed to decode JSON request body: %v", e)
		}
	}
	return params, query, body
}

type K8sEnv struct {
	Cluster    string
	Namespace  string
	Pod        string
	Container  string
	KubeConfig string
}

func fetchK8sEnv(ctx context.Context, w http.ResponseWriter, r *http.Request) (*K8sEnv, error) {
	params, _, body := fetchEnv(ctx, w, r)
	cluster, _ := body.GetString("cluster")
	if cluster == "" {
		cluster = "default"
	}
	namespace, _ := body.GetString("namespace")
	if namespace == "" {
		namespace = "default"
	}
	podName := params["<podName>"]
	container, _ := body.GetString("container")
	adminSession := auth.GetAdminSession(o.Options.Region, "")

	query := jsonutils.NewDict()
	query.Add(jsonutils.NewString(namespace), "namespace")
	query.Add(jsonutils.NewString(cluster), "cluster")
	obj, err := k8s.Pods.Get(adminSession, podName, query)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, httperrors.NewNotFoundError("Not found pod %q", podName)
	}

	data := jsonutils.NewDict()
	data.Add(jsonutils.JSONTrue, "directly")
	ret, err := k8s.Clusters.PerformAction(adminSession, cluster, "generate-kubeconfig", data)
	if err != nil {
		return nil, err
	}
	conf, err := ret.GetString("kubeconfig")
	if err != nil {
		return nil, httperrors.NewNotFoundError("Not found cluster %q kubeconfig", cluster)
	}
	f, err := ioutil.TempFile("", "kubeconfig-")
	if err != nil {
		return nil, fmt.Errorf("Save kubeconfig error: %v", err)
	}
	f.WriteString(conf)
	defer f.Close()

	return &K8sEnv{
		Cluster:    cluster,
		Namespace:  namespace,
		Pod:        podName,
		Container:  container,
		KubeConfig: f.Name(),
	}, nil
}

type CloudEnv struct {
	ClientSessin *mcclient.ClientSession
	Params       map[string]string
	Query        jsonutils.JSONObject
	Body         jsonutils.JSONObject
	Ctx          context.Context
}

func fetchCloudEnv(ctx context.Context, w http.ResponseWriter, r *http.Request) (*CloudEnv, error) {
	params, query, body := fetchEnv(ctx, w, r)
	userCred := auth.FetchUserCredential(ctx)
	if userCred == nil {
		return nil, httperrors.NewUnauthorizedError("No token founded")
	}
	s := auth.Client().NewSession(o.Options.Region, "", "internal", userCred, "v2")
	return &CloudEnv{
		ClientSessin: s,
		Params:       params,
		Query:        query,
		Body:         body,
		Ctx:          ctx,
	}, nil
}

func handleK8sCommand(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	cmdFactory func(kubeconfig, namespace, pod, container string) command.ICommand,
) {
	env, err := fetchK8sEnv(ctx, w, r)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}

	cmd := cmdFactory(env.KubeConfig, env.Namespace, env.Pod, env.Container)
	handleCommandSession(cmd, w)
}

func handleK8sShell(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	handleK8sCommand(ctx, w, r, command.NewPodBashCommand)
}

func handleK8sLog(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	handleK8sCommand(ctx, w, r, command.NewPodLogCommand)
}

func handleServerShell(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	env, err := fetchCloudEnv(ctx, w, r)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	serverId := env.Params["<id>"]
	ret, err := modules.Servers.Get(env.ClientSessin, serverId, jsonutils.Marshal(map[string]bool{"with_meta": true}))
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	info := command.SSHInfo{}
	err = ret.Unmarshal(&info)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	cmd, err := command.NewSSHtoolSolCommand(&info)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	handleCommandSession(cmd, w)
}

func handleBaremetalShell(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	env, err := fetchCloudEnv(ctx, w, r)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	hostId := env.Params["<id>"]
	ret, err := modules.Hosts.GetSpecific(env.ClientSessin, hostId, "ipmi", nil)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	info := command.IpmiInfo{}
	err = ret.Unmarshal(&info)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	cmd, err := command.NewIpmitoolSolCommand(&info)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	handleCommandSession(cmd, w)
}

func handleCommandSession(cmd command.ICommand, w http.ResponseWriter) {
	cmdSession, err := session.Manager.Save(cmd)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	data := jsonutils.NewDict()
	params, err := cmdSession.GetConnectParams()
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	data.Add(jsonutils.NewString(params), "connect_params")
	data.Add(jsonutils.NewString(cmdSession.Id), "session")
	sendJSON(w, data)
}

func sendJSON(w http.ResponseWriter, body jsonutils.JSONObject) {
	ret := jsonutils.NewDict()
	if body != nil {
		ret.Add(body, "webconsole")
	}
	appsrv.SendJSON(w, ret)
}
