package pageservices

import (
	"sync"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servicehealth"
)

type reportedConnection int

const (
	connectionUnreported reportedConnection = iota
	connectionUp
	connectionLost
)

type connectionWatch struct {
	observer         Observer
	pagescrapeHealth *servicehealth.Health
	pagerwiHealth    *servicehealth.Health
	mutex            sync.Mutex
	reported         reportedConnection
	ended            bool
}

func newConnectionWatch(observer Observer) *connectionWatch {
	watch := &connectionWatch{observer: observer}
	watch.pagescrapeHealth = servicehealth.ReportingTo(watch.healthChanged)
	watch.pagerwiHealth = servicehealth.ReportingTo(watch.healthChanged)
	return watch
}

func (w *connectionWatch) healthChanged() {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	if w.ended {
		return
	}
	switch {
	case !w.pagescrapeHealth.Connected() || !w.pagerwiHealth.Connected():
		w.report(connectionLost, w.observer.BrokerConnectionLost)
	case w.pagescrapeHealth.Healthy() && w.pagerwiHealth.Healthy():
		w.report(connectionUp, w.observer.ServicesConnected)
	}
}

func (w *connectionWatch) report(connection reportedConnection, tell func()) {
	if w.reported == connection {
		return
	}
	w.reported = connection
	tell()
}

func (w *connectionWatch) end() {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.ended = true
}
