package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "os"
    "path/filepath"

    "volcano-job-analyzer/cmd/analyze"
    "volcano-job-analyzer/cmd/validate"
    "volcano-job-analyzer/pkg/utils"

    "go.uber.org/zap"
)

var (
    configPath   = flag.String("config", "configs/analyzer-config.yaml", "Path to configuration file")
    logLevel     = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
    outputFormat = flag.String("output", "table", "Output format (table, json, yaml)")
    version      = flag.Bool("version", false, "Show version information")
)

const (
    appVersion = "v1.0.0"
    appName    = "volcano-job-analyzer"
)

func main() {
    flag.Parse()

    if *version {
        fmt.Printf("%s %s\n", appName, appVersion)
        os.Exit(0)
    }

    // Initialize logger
    logger, err := utils.NewLogger(*logLevel)
    if err != nil {
        log.Fatalf("Failed to initialize logger: %v", err)
    }

    // Load configuration
    config, err := utils.LoadConfig(*configPath)
    if err != nil {
        logger.Error("Failed to load configuration", zap.Error(err), zap.String("path", *configPath))
        os.Exit(1)
    }

    if len(flag.Args()) < 2 {
        printUsage()
        os.Exit(1)
    }

    command := flag.Args()[0]
    ctx := context.Background()

    switch command {
    case "analyze":
        if err := analyze.Execute(ctx, flag.Args()[1:], config, logger, *outputFormat); err != nil {
            logger.Error("Analysis failed", zap.Error(err))
            os.Exit(1)
        }
    case "validate":
        if err := validate.Execute(ctx, flag.Args()[1:], config, logger); err != nil {
            logger.Error("Validation failed", zap.Error(err))
            os.Exit(1)
        }
    default:
        logger.Error("Unknown command", zap.String("command", command))
        printUsage()
        os.Exit(1)
    }
}

func printUsage() {
    fmt.Fprintf(os.Stderr, "Usage: %s [flags] <command> <args>\n", filepath.Base(os.Args[0]))
    fmt.Fprintf(os.Stderr, "\nCommands:\n")
    fmt.Fprintf(os.Stderr, "  analyze <job-file>     Analyze Volcano Job resource requirements\n")
    fmt.Fprintf(os.Stderr, "  validate <job-file>    Validate Volcano Job specification\n")
    fmt.Fprintf(os.Stderr, "\nFlags:\n")
    flag.PrintDefaults()
    fmt.Fprintf(os.Stderr, "\nExamples:\n")
    fmt.Fprintf(os.Stderr, "  %s analyze examples/pytorch-job.yaml\n", filepath.Base(os.Args[0]))
    fmt.Fprintf(os.Stderr, "  %s --output json analyze examples/tensorflow-job.yaml\n", filepath.Base(os.Args[0]))
    fmt.Fprintf(os.Stderr, "  %s validate examples/complex-job.yaml\n", filepath.Base(os.Args[0]))
}