package observer

import (
	"runtime"
	"runtime/debug"
	"time"

	"github.com/coalalib/coalago/common"
	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/pools"
	"github.com/coalalib/coalago/resource"
	logging "github.com/op/go-logging"
)

var (
	log                 = logging.MustGetLogger("coala.Observer")
	metricObservers     int64
	metricObservingTime int64
)

const (
	DEFAULT_MAX_AGE = 30
)

type CoAPObserver struct {
	coala common.SenderIface
}

func StartObserver(coala common.SenderIface, pools *pools.AllPools) {
	obsrv := &CoAPObserver{
		coala: coala,
	}

	go obsrv.run(pools)
}

func (c *CoAPObserver) run(pools *pools.AllPools) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("CoAPcallback.run", "Logging the recovered error: ", r)
			stackBuf := debug.Stack()
			log.Error("CoAPcallback.run", "Runtime error: "+string(stackBuf))

			go c.run(pools)
		}
	}()

	balancerChan := make(chan bool, runtime.NumCPU())

	for {
		timerNow := time.Now()
		for _, obsrv := range pools.Observers.GetAll() {
			callback := obsrv.(*CoAPObserverCallback)
			cond := *callback.Condition

			n := time.Now().Unix() - callback.LastUpdate

			if n > int64(callback.MaxAge) {
				pools.Observers.Delete(callback.Key)
				continue
			}

			if cond(callback) && !callback.inProcess {
				go func() {
					balancerChan <- true
					ObserverExec(c.coala, pools, callback)
					<-balancerChan
				}()
			}
		}
		metricObservers = int64(pools.Observers.Count())
		metricObservingTime = time.Since(timerNow).Nanoseconds()
		time.Sleep(100 * time.Millisecond)
	}
}

func SendNotification(coala common.SenderIface, pools *pools.AllPools, code m.CoapCode, callback *CoAPObserverCallback, payload m.CoAPMessagePayload) {
	notification := m.NewCoAPMessage(m.CON, code)
	notification.Token = callback.RegisteredMessage.Token
	notification.Payload = payload

	// Send this Error Notification
	coala.Send(notification, callback.RegisteredMessage.Sender)
	pools.Observers.Delete(callback.Key)
}

func ObserverExec(coala common.SenderIface, pools *pools.AllPools, callback *CoAPObserverCallback) {
	callback.inProcess = true
	defer func() {
		callback.inProcess = false
	}()

	handlerResult, isContinue := getHandlerResult(coala, pools, callback)
	if !isContinue {
		return
	}

	notification := NewObserverNotification(handlerResult, callback)

	log.Debug("Sending observe notification with Token: " + notification.GetTokenString())

	// Send the Notification!
	respMsg, err := coala.Send(notification, callback.RegisteredMessage.Sender)

	if err != nil {
		log.Error("Notification is error, remove", err)
		pools.Observers.Delete(callback.Key)
		return
	}
	if respMsg != nil {
		if respMsg.Type == m.RST {
			log.Debug("Notification is reset, remove")
			pools.Observers.Delete(callback.Key)
			return
		}
		callback.LastUpdate = time.Now().Unix()
	}

	return
}

func getHandlerResult(coala common.SenderIface, pools *pools.AllPools, callback *CoAPObserverCallback) (handlerResult *resource.CoAPResourceHandlerResult, isContinue bool) {
	resources := coala.GetResourcesForPathAndMethod(callback.RegisteredMessage.GetURIPath(), callback.RegisteredMessage.GetMethod())
	if len(resources) == 0 {
		log.Debug("Length resources is 0")
		SendNotification(coala, pools, m.CoapCodeNotFound, callback, nil)
		return nil, false
	}

	handlerResult = resources[0].Handler(callback.RegisteredMessage)

	if handlerResult == nil {
		log.Debug("Handler result is NIL")
		SendNotification(coala, pools, m.CoapCodeInternalServerError, callback, nil)
		return nil, false
	}

	// 128 and 160 this is errors code from CoAPCore
	if handlerResult.Code >= 128 && handlerResult.Code < 160 {
		log.Debug("Handler result code is error")
		SendNotification(coala, pools, handlerResult.Code, callback, handlerResult.Payload)
		return nil, false
	}
	return handlerResult, true
}

func NewObserverNotification(handlerResult *resource.CoAPResourceHandlerResult, callback *CoAPObserverCallback) *m.CoAPMessage {
	notification := m.NewCoAPMessage(m.CON, handlerResult.Code)
	notification.Payload = handlerResult.Payload
	notification.Token = callback.RegisteredMessage.Token
	notification.AddOption(m.OptionObserve, callback.getNextOrdering())

	log.Debug("Sending notification", notification.GetTokenString(), notification.MessageID)
	notification.AddOption(m.OptionMaxAge, callback.MaxAge)
	return notification
}
