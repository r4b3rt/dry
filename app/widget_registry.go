package app

import (
	"sync"

	"github.com/gizak/termui"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/appui/swarm"
	"github.com/moncho/dry/ui"
)

type widget interface {
	termui.Bufferer
	Mount() error
	Unmount() error
	Name() string
}

//widgetRegistry holds references to two types of widgets:
// * widgets that hold information that does not change or widgets
//   that hold information that is worth updating only when is changed.
//   These are all the widget tracked with a field in the struct.
// * a set of widgets to be rendered on the next rendering phase.
//
type widgetRegistry struct {
	ContainerList *appui.ContainersWidget
	ContainerMenu *appui.ContainerMenuWidget
	DiskUsage     *appui.DockerDiskUsageRenderer
	DockerInfo    *appui.DockerInfo
	ImageList     *appui.DockerImagesWidget
	MessageBar    *ui.ExpiringMessageWidget
	Monitor       *appui.Monitor
	Networks      *appui.DockerNetworksWidget
	Nodes         *swarm.NodesWidget
	NodeTasks     *swarm.NodeTasksWidget
	ServiceTasks  *swarm.ServiceTasksWidget
	ServiceList   *swarm.ServicesWidget
	Stacks        *swarm.StacksWidget
	StackTasks    *swarm.StacksTasksWidget
	Volumes       *appui.VolumesWidget
	sync.RWMutex
	widgets map[string]widget
}

func (wr *widgetRegistry) add(w widget) error {
	wr.Lock()
	defer wr.Unlock()
	err := w.Mount()
	if err == nil {
		wr.widgets[w.Name()] = w
	}
	return err
}

func (wr *widgetRegistry) remove(w widget) error {
	wr.Lock()
	defer wr.Unlock()
	delete(wr.widgets, w.Name())
	return w.Unmount()
}

func (wr *widgetRegistry) activeWidgets() []widget {
	wr.RLock()
	defer wr.RUnlock()
	widgets := make([]widget, len(wr.widgets))
	i := 0
	for _, widget := range wr.widgets {
		widgets[i] = widget
		i++
	}
	return widgets
}
func (wr *widgetRegistry) reload() {

}
