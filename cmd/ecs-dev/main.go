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

const version = "0.1.0"

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
				dockerClient.Close()
			}
		},
	}

	rootCmd.AddCommand(runCmd())
	rootCmd.AddCommand(psCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runCmd() *cobra.Command {
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

	return cmd
}

func psCmd() *cobra.Command {
	var showAll bool

	cmd := &cobra.Command{
		Use:   "ps",
		Short: "List running tasks",
		Long:  "Display all tasks managed by ecs-dev",
		RunE: func(cmd *cobra.Command, args []string) error {
			tasks := taskManager.ListTasks()

			if len(tasks) == 0 {
				fmt.Println("No tasks running")
				return nil
			}

			fmt.Printf("%-25s %-20s %-15s %-10s %s\n", "TASK ID", "FAMILY", "STATUS", "CONTAINERS", "STARTED")
			fmt.Println("--------------------------------------------------------------------------------------")

			for _, t := range tasks {
				if !showAll && t.Status == "STOPPED" {
					continue
				}

				fmt.Printf("%-25s %-20s %-15s %-10d %s\n",
					t.ID,
					t.Family,
					t.Status,
					len(t.Containers),
					t.StartedAt.Format("2006-01-02 15:04:05"),
				)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&showAll, "all", "a", false, "Show all tasks including stopped")

	return cmd
}
