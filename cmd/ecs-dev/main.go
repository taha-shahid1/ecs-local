package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/taha-shahid1/ecs-local/pkg/docker"
	"github.com/taha-shahid1/ecs-local/pkg/parser"
	"github.com/taha-shahid1/ecs-local/pkg/task"
)

const (
	version           = "0.1.0"
	taskStatusRunning = "RUNNING"
	taskStatusStopped = "STOPPED"
)

var (
	dockerClient *docker.Client
	taskManager  *task.Manager
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "ecs-dev",
		Short:   "ECS Local Development Tool",
		Long:    "Run and test ECS task definitions locally using Docker",
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Initialize Docker client for all commands except help/version
			if cmd.Name() == "help" || cmd.Name() == "version" {
				return nil
			}

			var err error
			dockerClient, err = docker.NewClient()
			if err != nil {
				return fmt.Errorf("failed to connect to Docker: %w", err)
			}

			taskManager = task.NewManager(dockerClient)
			return nil
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if dockerClient != nil {
				_ = dockerClient.Close()
			}
		},
	}

	rootCmd.AddCommand(runCmd())
	rootCmd.AddCommand(psCmd())
	rootCmd.AddCommand(logsCmd())
	rootCmd.AddCommand(stopCmd())
	rootCmd.AddCommand(rmCmd())
	rootCmd.AddCommand(execCmd())
	rootCmd.AddCommand(depsCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runCmd() *cobra.Command {
	var noCascade bool
	var dependencyTimeout int

	cmd := &cobra.Command{
		Use:   "run <task-definition.json>",
		Short: "Run a task from a task definition file",
		Long:  "Parse an ECS task definition and run it locally using Docker",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskDefFile := args[0]

			// Parse task definition
			fmt.Printf("Parsing task definition: %s\n", taskDefFile)
			taskDef, err := parser.ParseTaskDefinition(taskDefFile)
			if err != nil {
				return fmt.Errorf("failed to parse task definition: %w", err)
			}

			fmt.Printf("Task family: %s\n", taskDef.Family)
			fmt.Printf("Containers: %d\n\n", len(taskDef.ContainerDefinitions))

			// Configure cascade failure behavior
			taskManager.SetCascadeOnFailure(!noCascade)
			if noCascade {
				fmt.Println("Cascade failure disabled - dependent containers will continue running if dependencies fail")
			}

			// Run the task
			ctx := context.Background()
			runningTask, err := taskManager.RunTask(ctx, taskDef)
			if err != nil {
				return fmt.Errorf("failed to run task: %w", err)
			}

			fmt.Printf("\n✓ Task started successfully\n")
			fmt.Printf("Task ID: %s\n", runningTask.ID)
			fmt.Printf("Status: %s\n", runningTask.Status)
			fmt.Printf("Containers:\n")
			for name, id := range runningTask.Containers {
				fmt.Printf("  - %s: %s\n", name, id[:12])
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&noCascade, "no-cascade", false, "Don't stop dependent containers when a dependency fails")
	cmd.Flags().IntVar(&dependencyTimeout, "dependency-timeout", 0, "Timeout in seconds for dependency conditions (0 = use defaults)")

	return cmd
}

func psCmd() *cobra.Command {
	var showAll bool
	var showDeps bool

	cmd := &cobra.Command{
		Use:   "ps",
		Short: "List running tasks",
		Long:  "Display all tasks managed by ecs-dev",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			tasks := taskManager.ListTasks()

			if len(tasks) == 0 {
				fmt.Println("No tasks running")
				return nil
			}

			if showDeps {
				// Show detailed dependency information
				for _, t := range tasks {
					if !showAll && t.Status == taskStatusStopped {
						continue
					}
					info, err := taskManager.GetTaskDependencyInfo(t.ID)
					if err != nil {
						fmt.Printf("Error getting dependency info for task %s: %v\n", t.ID, err)
						continue
					}
					fmt.Println(info.FormatDependencyGraph())
					fmt.Println()
				}
				return nil
			}

			fmt.Printf("%-25s %-20s %-15s %-10s %-15s %s\n", "TASK ID", "FAMILY", "STATUS", "CONTAINERS", "HEALTH", "STARTED")
			fmt.Println("-------------------------------------------------------------------------------------------------------")

			for _, t := range tasks {
				if !showAll && t.Status == taskStatusStopped {
					continue
				}

				healthStatus := getTaskHealthStatus(ctx, t)

				fmt.Printf("%-25s %-20s %-15s %-10d %-15s %s\n",
					t.ID,
					t.Family,
					t.Status,
					len(t.Containers),
					healthStatus,
					t.StartedAt.Format("2006-01-02 15:04:05"),
				)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&showAll, "all", "a", false, "Show all tasks including stopped")
	cmd.Flags().BoolVarP(&showDeps, "deps", "d", false, "Show detailed dependency information")

	return cmd
}

func getTaskHealthStatus(ctx context.Context, t *task.Task) string {
	if len(t.Containers) == 0 {
		return ""
	}

	healthStatuses := make(map[string]int)
	hasHealthChecks := false

	for _, containerID := range t.Containers {
		health, err := dockerClient.GetContainerHealth(ctx, containerID)
		if err != nil || health == "none" {
			continue
		}
		hasHealthChecks = true
		healthStatuses[health]++
	}

	if !hasHealthChecks {
		return ""
	}

	if healthStatuses["unhealthy"] > 0 {
		return "\033[31munhealthy\033[0m"
	}
	if healthStatuses["starting"] > 0 {
		return "\033[33mstarting\033[0m"
	}
	if healthStatuses["healthy"] > 0 {
		return "\033[32mhealthy\033[0m"
	}

	return ""
}

func logsCmd() *cobra.Command {
	var (
		follow     bool
		timestamps bool
		tail       string
		taskID     string
	)

	cmd := &cobra.Command{
		Use:   "logs <container-name>",
		Short: "View container logs",
		Long:  "Stream logs from a container. Use --task to view logs from all containers in a task.",
		Args: func(cmd *cobra.Command, args []string) error {
			if taskID != "" {
				return nil
			}
			if len(args) != 1 {
				return fmt.Errorf("requires container-name argument or --task flag")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			if taskID != "" {
				// Get task and stream logs from all containers
				task, err := taskManager.GetTask(taskID)
				if err != nil {
					return fmt.Errorf("task not found: %s", taskID)
				}

				containerIDs := make([]string, 0, len(task.Containers))
				for _, id := range task.Containers {
					containerIDs = append(containerIDs, id)
				}

				fmt.Printf("Streaming logs from %d container(s) in task %s\n", len(containerIDs), taskID)
				fmt.Println("Press Ctrl+C to exit")
				fmt.Println()

				return dockerClient.StreamMultipleLogs(ctx, containerIDs, follow)
			}

			// Single container logs
			containerName := args[0]

			// Try to find container by name in all tasks
			var containerID string
			for _, task := range taskManager.ListTasks() {
				if id, exists := task.Containers[containerName]; exists {
					containerID = id
					break
				}
			}

			if containerID == "" {
				return fmt.Errorf("container not found: %s", containerName)
			}

			opts := docker.LogOptions{
				Follow:     follow,
				Timestamps: timestamps,
				Tail:       tail,
			}

			return dockerClient.StreamLogs(ctx, containerID, opts)
		},
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output")
	cmd.Flags().BoolVarP(&timestamps, "timestamps", "t", false, "Show timestamps")
	cmd.Flags().StringVar(&tail, "tail", "all", "Number of lines to show from the end")
	cmd.Flags().StringVar(&taskID, "task", "", "Show logs from all containers in a task")

	return cmd
}

func stopCmd() *cobra.Command {
	var stopAll bool

	cmd := &cobra.Command{
		Use:   "stop [task-id]",
		Short: "Stop a running task",
		Long:  "Stop all containers in a task. Use --all to stop all tasks.",
		Args: func(cmd *cobra.Command, args []string) error {
			if stopAll {
				return nil
			}
			if len(args) != 1 {
				return fmt.Errorf("requires task-id argument or --all flag")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			if stopAll {
				tasks := taskManager.ListTasks()
				if len(tasks) == 0 {
					fmt.Println("No tasks to stop")
					return nil
				}

				fmt.Printf("Stopping %d task(s)...\n", len(tasks))
				for _, t := range tasks {
					if t.Status == taskStatusStopped {
						continue
					}

					err := taskManager.StopTask(ctx, t.ID)
					if err != nil {
						fmt.Printf("Warning: failed to stop task %s: %v\n", t.ID, err)
					} else {
						fmt.Printf("✓ Stopped task: %s\n", t.ID)
					}
				}
				return nil
			}

			taskID := args[0]
			err := taskManager.StopTask(ctx, taskID)
			if err != nil {
				return fmt.Errorf("failed to stop task: %w", err)
			}

			fmt.Printf("✓ Task stopped: %s\n", taskID)
			return nil
		},
	}

	cmd.Flags().BoolVar(&stopAll, "all", false, "Stop all running tasks")

	return cmd
}

func rmCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "rm <task-id>",
		Short: "Remove a stopped task",
		Long:  "Remove all containers associated with a task. Use -f to force remove running tasks.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			taskID := args[0]

			// Check if task exists
			task, err := taskManager.GetTask(taskID)
			if err != nil {
				return fmt.Errorf("task not found: %s", taskID)
			}

			// If not forcing and task is running, warn
			if !force && task.Status == taskStatusRunning {
				return fmt.Errorf("task is still running. Stop it first or use -f to force remove")
			}

			// Stop if running
			if task.Status == taskStatusRunning {
				fmt.Printf("Stopping task %s...\n", taskID)
				err = taskManager.StopTask(ctx, taskID)
				if err != nil {
					return fmt.Errorf("failed to stop task: %w", err)
				}
			}

			// Remove
			err = taskManager.RemoveTask(ctx, taskID)
			if err != nil {
				return fmt.Errorf("failed to remove task: %w", err)
			}

			fmt.Printf("✓ Task removed: %s\n", taskID)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force remove running task")

	return cmd
}

func execCmd() *cobra.Command {
	var (
		interactive bool
		user        string
		workdir     string
	)

	cmd := &cobra.Command{
		Use:   "exec <container-name> <command> [args...]",
		Short: "Execute a command in a running container",
		Long:  "Run a command inside a running container. Use -i for interactive shell.",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			containerName := args[0]
			command := args[1:]

			var containerID string
			found := false

			for _, t := range taskManager.ListTasks() {
				if t.Status != taskStatusRunning {
					continue
				}

				if id, exists := t.Containers[containerName]; exists {
					containerID = id
					found = true
					break
				}
			}

			if !found {
				for _, t := range taskManager.ListTasks() {
					for cName, cID := range t.Containers {
						fullContainerName := fmt.Sprintf("%s-%s", t.ID, cName)
						if fullContainerName == containerName {
							containerID = cID
							found = true
							break
						}
					}
					if found {
						break
					}
				}

				if !found {
					id, err := dockerClient.GetContainerID(ctx, containerName)
					if err != nil {
						return fmt.Errorf("container not found: %s", containerName)
					}
					containerID = id
				}
			}

			if interactive {
				execConfig := docker.ExecConfig{
					ContainerID: containerID,
					Cmd:         command,
					Interactive: true,
					Tty:         true,
					WorkingDir:  workdir,
					User:        user,
				}

				return dockerClient.ExecInteractive(ctx, execConfig)
			}

			exitCode, err := dockerClient.ExecWithIO(ctx, containerID, command, docker.ContainerExecAttachOptions{
				Stdout: os.Stdout,
				Stderr: os.Stderr,
			})

			if err != nil {
				return err
			}

			if exitCode != 0 {
				os.Exit(exitCode)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Keep STDIN open and allocate a pseudo-TTY")
	cmd.Flags().StringVarP(&user, "user", "u", "", "Username or UID")
	cmd.Flags().StringVarP(&workdir, "workdir", "w", "", "Working directory inside the container")

	return cmd
}

func depsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deps [task-id]",
		Short: "Show dependency information for tasks",
		Long:  "Display dependency graph and waiting status for containers in a task",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tasks := taskManager.ListTasks()

			if len(tasks) == 0 {
				fmt.Println("No tasks running")
				return nil
			}

			if len(args) == 0 {
				// Show dependency info for all tasks
				for _, t := range tasks {
					info, err := taskManager.GetTaskDependencyInfo(t.ID)
					if err != nil {
						fmt.Printf("Error getting dependency info for task %s: %v\n", t.ID, err)
						continue
					}
					fmt.Println(info.FormatDependencyGraph())
					fmt.Println()
				}
			} else {
				// Show dependency info for specific task
				taskID := args[0]
				info, err := taskManager.GetTaskDependencyInfo(taskID)
				if err != nil {
					return fmt.Errorf("failed to get dependency info: %w", err)
				}
				fmt.Println(info.FormatDependencyGraph())
			}

			return nil
		},
	}

	return cmd
}
