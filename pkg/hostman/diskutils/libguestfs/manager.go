package libguestfs

import (
	"sync"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/hostman/diskutils/libguestfs/guestfish"
)

type ErrorFish string

func (e ErrorFish) Error() string {
	return string(e)
}

const (
	DEFAULT_FISHCAGE_SIZE = 1

	ErrFishsMismatch = ErrorFish("fishs mismatch")
	ErrFishsWorking  = ErrorFish("fishs working")
)

var guestfsManager *GuestfsManager

func Init(count int) {
	if guestfsManager == nil {
		guestfsManager = NewGuestfsManager(count)
		time.AfterFunc(time.Minute*3, guestfsManager.makeFishHappy)
	}
}

type GuestfsManager struct {
	fishMaximum      int
	happyFishCount   int
	workingFishCount int

	fishs    map[*guestfish.Guestfish]bool
	fishChan chan *guestfish.Guestfish
	fishlock sync.Mutex

	lastTimeFishing time.Time
}

func NewGuestfsManager(count int) *GuestfsManager {
	if count < 1 {
		count = DEFAULT_FISHCAGE_SIZE
	}
	return &GuestfsManager{
		fishMaximum: count,
		fishs:       make(map[*guestfish.Guestfish]bool, count),
		fishChan:    make(chan *guestfish.Guestfish, count),
	}
}

func (m *GuestfsManager) AcquireFish() (*guestfish.Guestfish, error) {
	m.fishlock.Lock()
	defer m.fishlock.Unlock()

	m.lastTimeFishing = time.Now()
	fish, err := m.acquireFish()
	if err == ErrFishsWorking {
		return m.waitingFishFinish(), nil
	} else if err != nil {
		return nil, err
	}
	return fish, nil
}

func (m *GuestfsManager) acquireFish() (*guestfish.Guestfish, error) {
	if m.happyFishCount == 0 {
		if 0 < m.workingFishCount && m.workingFishCount < m.fishMaximum {
			fish, err := guestfish.newGuestfish()
			if err != nil {
				return nil, err
			}
			m.fishs[fish] = true
			m.workingFishCount++
			return fish, nil
		} else {
			return nil, ErrFishsWorking
		}
	} else {
		for fish, working := range m.fishs {
			if !working {
				m.fishs[fish] = true
				m.workingFishCount++
				m.happyFishCount--
				return fish, nil
			}
		}
		return nil, ErrFishsMismatch
	}
}

func (m *GuestfsManager) waitingFishFinish() *guestfish.Guestfish {
	select {
	case fish := <-m.fishChan:
		return fish
	}
}

func (m *GuestfsManager) ReleaseFish(fish *guestfish.Guestfish) {
	err := m.washfish(fish)
	m.fishlock.Lock()
	defer m.fishlock.Unlock()
	if err != nil {
		log.Errorf("wash fish failed: %s", err)
		err = fish.quit()
		if err != nil {
			log.Errorf("fish quit failed: %s", err)
		}
		delete(m.fishs, fish)
		m.workingFishCount--
	}
	m.fishChan <- fish
}

func (m *GuestfsManager) washfish(fish *guestfish.Guestfish) error {
	return fish.removeDrive()
}

func (m *GuestfsManager) makeFishHappy() {
	defer time.AfterFunc(time.Minute*10, m.makeFishHappy)
	m.fishlock.Lock()
	defer m.fishlock.Unlock()
	if m.lastTimeFishing.IsZero() || time.Now().Sub(m.lastTimeFishing) < time.Minute*10 {
		return
	}

Loop:
	for {
		select {
		case fish := <-m.fishChan:
			m.fishs[fish] = false
			m.happyFishCount++
			m.workingFishCount--
		default:
			break Loop
		}
	}

	for fish, working := range m.fishs {
		if !working {
			err := fish.quit()
			if err != nil {
				log.Errorf("fish quit failed: %s", err)
			}
			m.happyFishCount--
			delete(m.fishs, fish)
		}
	}
}
