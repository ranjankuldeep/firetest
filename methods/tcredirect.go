package methods

import (
	"ranjankuldeep/test/netlink"

	"github.com/weaveworks/ignite/pkg/logs"
)

func SetUpSandBoxNetwork(nsPath string, uid, gid int) error {
	net := netlink.DefaultNetlinkOps()
	taskTap := Task{
		Execute: func() error {
			logs.Logger.Infof("Attaching Tap Device %s To Sandbox %s", "tap0", nsPath)
			if err := net.AttachTap(nsPath, "tap0", 1500, uid, gid); err != nil {
				return err
			}
			return nil
		},
		Cleanup: func() error {
			logs.Logger.Info("Automatic Cleanup of Tap Devices")
			return nil
		},
	}
	taskTCRedirect := Task{
		Execute: func() error {
			logs.Logger.Infof("Adding tc redirect between %s:%s", "eth0", "tap0")
			sandboxMainIface := "eth0"
			sandboxTapIface := "tap0"

			if err := netlink.DefaultNetlinkOps().AddTcRedirect(nsPath, sandboxMainIface, sandboxTapIface); err != nil {
				logs.Logger.Errorf("Failed to setup tc redirect %v", err)
				return err
			}
			return nil
		},
		Cleanup: func() error {
			return nil
		},
	}
	tasks := []Task{taskTap, taskTCRedirect}
	if err := executeTasks(tasks); err != nil {
		logs.Logger.Errorf("Failed to execute all tasks: %v\n", err)
		return err
	}
	logs.Logger.Info("Sandbox Network Setup Succesfully")
	return nil
}

type Task struct {
	Execute func() error
	Cleanup func() error
}

func executeTasks(tasks []Task) error {
	var executedTasks []Task

	for _, task := range tasks {
		if err := task.Execute(); err != nil {
			logs.Logger.Errorf("Task failed with error: %v\n", err)
			for i := len(executedTasks) - 1; i >= 0; i-- {
				executedTasks[i].Cleanup()
			}
			return err
		}
		executedTasks = append(executedTasks, task)
	}
	return nil
}
