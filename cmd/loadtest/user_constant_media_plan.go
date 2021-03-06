package main

// The user constant test plan has 10 users per room. Users take a random
// break after logging in. Messages are sent on a loop with a random
// Message Break inbetween. Image messages are also sent.

import (
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"time"

	l "github.com/mattermost/mattermost-load-test/lib"
	p "github.com/mattermost/mattermost-load-test/platform"
)

// UserConstantMediaPlan - try out the test plan interface
type UserConstantMediaPlan struct {
	id              int
	activityChannel chan<- l.Activity
	stopChannel     chan bool
	mm              p.Platform
}

// Generator sets up & exports the channels
func (tp UserConstantMediaPlan) Generator(id int, activityChannel chan<- l.Activity) l.TestPlan {
	newPlan := new(UserConstantMediaPlan)
	newPlan.id = id
	newPlan.activityChannel = activityChannel
	newPlan.mm = p.GeneratePlatform(Config.PlatformURL)
	return newPlan
}

// Start is a long running function that should only quit on error
func (tp *UserConstantMediaPlan) Start() bool {

	defer tp.PanicCheck()

	userEmail := GeneratePlatformEmail(tp.id)
	userPassword := GeneratePlatformPass(tp.id)

	// Login User
	err := tp.mm.Login(userEmail, userPassword)
	if err != nil {
		tp.handleError(err, "Login Failed", false)
		tp.registerLaunchFail()
		return false
	}

	rand.Seed(int64(tp.id))
	rando := rand.Intn(Config.LoginBreak)
	sleepDuration := time.Duration(rando) * time.Second
	time.Sleep(sleepDuration)
	tp.registerActive()

	// Initial Load
	err = tp.mm.InitialLoad()
	if err != nil {
		return tp.handleError(err, "Initial Load Failed", true)
	}

	// Team Lookup Load
	_, err = tp.mm.FindTeam(Config.TeamName, true)
	if err != nil {
		return tp.handleError(err, "Team Lookup Failed", true)
	}

	channelExtension := tp.id / 10
	userChannel := fmt.Sprintf("%v%v", Config.TestChannel, channelExtension)

	channel, err := tp.mm.GetChannel(userChannel)
	if err != nil {
		return tp.handleError(err, "Create/Get Channel Failed", true)
	}

	for {
		select {
		case <-tp.stopChannel:
			return false
		default:
			if RandomChoice(Config.MediaPercent) {
				upload, upErr := tp.mm.UploadRandomImage(channel, p.RandomMessage{})
				if upErr != nil && !reflect.ValueOf(err).IsNil() {
					tp.handleError(upErr, "Failed on uploading random image", false)
					continue
				}
				err = tp.mm.SendAttachment(channel, "test media message", upload.Filenames, "")
				if err != nil && !reflect.ValueOf(err).IsNil() {
					tp.handleError(err, "Media Message Send Failed", false)
					continue
				}
			} else {
				message := p.RandomMessage{}.Plain()
				err = tp.mm.SendMessage(channel, message, "")
				if err != nil && !reflect.ValueOf(err).IsNil() {
					tp.handleError(err, "Message Send Failed", false)
					continue
				}
			}
			tp.threadSendMessage()
			time.Sleep(time.Second * time.Duration(rand.Intn(Config.MessageBreak)))
		}
	}

}

// Stop takes the result of start(), and can change return
// respond true if the thread should restart, false otherwise
func (tp *UserConstantMediaPlan) Stop() {
	if tp.stopChannel != nil {
		tp.stopChannel <- true
	}
}

// GlobalSetup will run before the test plan. It will spin up a basic test plan
// from the Generator and will not be reused.
func (tp *UserConstantMediaPlan) GlobalSetup() (err error) {
	return nil
}

// PanicCheck will check for panics, used as a defer in test plan
func (tp *UserConstantMediaPlan) PanicCheck() {
	if r := recover(); r != nil {
		if Error != nil {
			Error.Printf("ERROR ON WORKER: %v", r)
		} else {
			fmt.Printf("ERROR ON WORKER: %v", r)
		}
		switch x := r.(type) {
		case string:
			tp.handleError(errors.New(x), "Error caught unexpected (thread failed)", true)
		case error:
			tp.handleError(x, "Error caught unexpected (thread failed)", true)
		default:
			tp.handleError(errors.New("Unknown Panic"), "Error caught unexpected (thread failed)", true)
		}
	}
}

func (tp *UserConstantMediaPlan) registerActive() {
	tp.stopChannel = make(chan bool)
	tp.activityChannel <- l.Activity{
		Status:  l.StatusActive,
		ID:      tp.id,
		Message: "Thread active",
	}
}

func (tp *UserConstantMediaPlan) registerInactive() {
	tp.activityChannel <- l.Activity{
		Status:  l.StatusInactive,
		ID:      tp.id,
		Message: "Thread inactive",
	}
}

func (tp *UserConstantMediaPlan) registerLaunchFail() {
	tp.activityChannel <- l.Activity{
		Status:  l.StatusLaunchFailed,
		ID:      tp.id,
		Message: "Failed launch",
	}
}

func (tp *UserConstantMediaPlan) handleError(err error, message string, notify bool) bool {
	tp.activityChannel <- l.Activity{
		Status:  l.StatusError,
		ID:      tp.id,
		Message: message,
		Err:     err,
	}
	if notify {
		tp.registerInactive()
	}
	time.Sleep(time.Second * 5)
	return true
}

func (tp *UserConstantMediaPlan) threadSendMessage() {
	tp.activityChannel <- l.Activity{
		Status:  l.StatusAction,
		ID:      tp.id,
		Message: fmt.Sprintf("User %v sent a message", tp.id),
	}
}
