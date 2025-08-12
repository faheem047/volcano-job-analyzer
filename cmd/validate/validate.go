package validate

import (
	"context"
	"fmt"

	"volcano-job-analyzer/pkg/parser"
	"volcano-job-analyzer/pkg/utils"

	"go.uber.org/zap"
)

// Execute runs the validate command
func Execute(ctx context.Context, args []string, config *utils.Config, logger *zap.Logger) error {
	if len(args) == 0 {
		return fmt.Errorf("job file path is required")
	}

	jobFile := args[0]
	logger.Info("Validating job specification", zap.String("file", jobFile))

	// Parse and validate the job
	validator := parser.NewValidator()

	job, err := parser.ParseVolcanoJobFromFile(jobFile)
	if err != nil {
		return fmt.Errorf("failed to parse job file: %w", err)
	}

	// Validate the job specification
	if err := validator.ValidateJob(job); err != nil {
		logger.Error("Job validation failed", zap.Error(err))
		return fmt.Errorf("validation failed: %w", err)
	}

	// Additional semantic validations
	warnings := validator.CheckBestPractices(job)
	if len(warnings) > 0 {
		logger.Warn("Best practice warnings found")
		for _, warning := range warnings {
			fmt.Printf("WARNING: %s\n", warning)
		}
	}

	fmt.Printf("Job '%s' is valid!\n", job.Name)

	if len(warnings) == 0 {
		fmt.Println("No warnings found - follows all best practices.")
	} else {
		fmt.Printf("Found %d best practice warnings (see above).\n", len(warnings))
	}

	return nil
}
