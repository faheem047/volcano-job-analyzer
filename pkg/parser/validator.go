package parser

import (
	"fmt"
	"regexp"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	volcanov1alpha1 "volcano.sh/apis/pkg/apis/batch/v1alpha1"
)

// Validator provides comprehensive validation for Volcano Jobs
type Validator struct {
	config *ValidationConfig
}

// ValidationConfig holds validation configuration
type ValidationConfig struct {
	StrictMode               bool
	AllowEmptyResourceLimits bool
	MaxTasksPerJob           int
	MaxReplicasPerTask       int32
	RequiredLabels           []string
	ForbiddenImages          []string
}

// NewValidator creates a new validator with default configuration
func NewValidator() *Validator {
	return &Validator{
		config: &ValidationConfig{
			StrictMode:               false,
			AllowEmptyResourceLimits: true,
			MaxTasksPerJob:           20,
			MaxReplicasPerTask:       100,
			RequiredLabels:           []string{},
			ForbiddenImages:          []string{},
		},
	}
}

// NewValidatorWithConfig creates a validator with custom configuration
func NewValidatorWithConfig(config *ValidationConfig) *Validator {
	return &Validator{config: config}
}

// ValidateJob performs comprehensive validation of a Volcano Job
func (v *Validator) ValidateJob(job *volcanov1alpha1.Job) error {
	if err := v.validateMetadata(job); err != nil {
		return fmt.Errorf("metadata validation failed: %w", err)
	}

	if err := v.validateSpec(job); err != nil {
		return fmt.Errorf("spec validation failed: %w", err)
	}

	if err := v.validateTasks(job); err != nil {
		return fmt.Errorf("task validation failed: %w", err)
	}

	if err := v.validateResourceRequirements(job); err != nil {
		return fmt.Errorf("resource validation failed: %w", err)
	}

	if err := v.validateNetworking(job); err != nil {
		return fmt.Errorf("networking validation failed: %w", err)
	}

	if err := v.validateSecurity(job); err != nil {
		return fmt.Errorf("security validation failed: %w", err)
	}

	return nil
}

// validateMetadata validates job metadata
func (v *Validator) validateMetadata(job *volcanov1alpha1.Job) error {
	if job.Name == "" {
		return fmt.Errorf("job name is required")
	}

	// Validate name format
	nameRegex := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	if !nameRegex.MatchString(job.Name) {
		return fmt.Errorf("job name '%s' is invalid. Must match regex: %s", job.Name, nameRegex.String())
	}

	// Validate namespace if specified
	if job.Namespace != "" {
		if !nameRegex.MatchString(job.Namespace) {
			return fmt.Errorf("namespace '%s' is invalid", job.Namespace)
		}
	}

	// Check required labels
	for _, requiredLabel := range v.config.RequiredLabels {
		if job.Labels == nil || job.Labels[requiredLabel] == "" {
			return fmt.Errorf("required label '%s' is missing", requiredLabel)
		}
	}

	// Validate label values
	if job.Labels != nil {
		for key, value := range job.Labels {
			if err := v.validateLabelValue(key, value); err != nil {
				return fmt.Errorf("invalid label '%s': %w", key, err)
			}
		}
	}

	return nil
}

// validateSpec validates job specification
func (v *Validator) validateSpec(job *volcanov1alpha1.Job) error {
	if len(job.Spec.Tasks) == 0 {
		return fmt.Errorf("job must have at least one task")
	}

	if len(job.Spec.Tasks) > v.config.MaxTasksPerJob {
		return fmt.Errorf("job has %d tasks, maximum allowed is %d", len(job.Spec.Tasks), v.config.MaxTasksPerJob)
	}

	// Validate scheduler name if specified
	if job.Spec.SchedulerName != "" && job.Spec.SchedulerName != "volcano" {
		if v.config.StrictMode {
			return fmt.Errorf("scheduler name must be 'volcano', got '%s'", job.Spec.SchedulerName)
		}
	}

	// Validate queue if specified
	if job.Spec.Queue != "" {
		if !nameRegex.MatchString(job.Spec.Queue) {
			return fmt.Errorf("queue name '%s' is invalid", job.Spec.Queue)
		}
	}

	// Validate priority class if specified
	if job.Spec.PriorityClassName != "" {
		if !nameRegex.MatchString(job.Spec.PriorityClassName) {
			return fmt.Errorf("priority class name '%s' is invalid", job.Spec.PriorityClassName)
		}
	}

	return nil
}

// validateTasks validates all tasks in the job
func (v *Validator) validateTasks(job *volcanov1alpha1.Job) error {
	taskNames := make(map[string]bool)

	for i, task := range job.Spec.Tasks {
		// Check for duplicate task names
		if taskNames[task.Name] {
			return fmt.Errorf("duplicate task name '%s' found", task.Name)
		}
		taskNames[task.Name] = true

		if err := v.validateTask(task, i); err != nil {
			return fmt.Errorf("task '%s' validation failed: %w", task.Name, err)
		}
	}

	return nil
}

// validateTask validates a single task
func (v *Validator) validateTask(task volcanov1alpha1.TaskSpec, index int) error {
	if task.Name == "" {
		return fmt.Errorf("task[%d] name is required", index)
	}

	nameRegex := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	if !nameRegex.MatchString(task.Name) {
		return fmt.Errorf("task name '%s' is invalid", task.Name)
	}

	if task.Replicas <= 0 {
		return fmt.Errorf("task '%s' must have replicas > 0, got %d", task.Name, task.Replicas)
	}

	if task.Replicas > v.config.MaxReplicasPerTask {
		return fmt.Errorf("task '%s' has %d replicas, maximum allowed is %d", task.Name, task.Replicas, v.config.MaxReplicasPerTask)
	}

	// Validate pod template
	if err := v.validatePodTemplate(task.Template, task.Name); err != nil {
		return fmt.Errorf("pod template validation failed: %w", err)
	}

	return nil
}

// validatePodTemplate validates the pod template
func (v *Validator) validatePodTemplate(template corev1.PodTemplateSpec, taskName string) error {
	if len(template.Spec.Containers) == 0 {
		return fmt.Errorf("task '%s' must have at least one container", taskName)
	}

	containerNames := make(map[string]bool)
	for i, container := range template.Spec.Containers {
		// Check for duplicate container names
		if containerNames[container.Name] {
			return fmt.Errorf("duplicate container name '%s' in task '%s'", container.Name, taskName)
		}
		containerNames[container.Name] = true

		if err := v.validateContainer(container, taskName, i); err != nil {
			return fmt.Errorf("container '%s' validation failed: %w", container.Name, err)
		}
	}

	// Validate init containers if present
	for i, initContainer := range template.Spec.InitContainers {
		if err := v.validateContainer(initContainer, taskName, i); err != nil {
			return fmt.Errorf("init container '%s' validation failed: %w", initContainer.Name, err)
		}
	}

	// Validate volumes
	if err := v.validateVolumes(template.Spec.Volumes, taskName); err != nil {
		return fmt.Errorf("volume validation failed: %w", err)
	}

	return nil
}

// validateContainer validates a single container
func (v *Validator) validateContainer(container corev1.Container, taskName string, index int) error {
	if container.Name == "" {
		return fmt.Errorf("container[%d] in task '%s' must have a name", index, taskName)
	}

	if container.Image == "" {
		return fmt.Errorf("container '%s' in task '%s' must have an image", container.Name, taskName)
	}

	// Check forbidden images
	for _, forbidden := range v.config.ForbiddenImages {
		if strings.Contains(container.Image, forbidden) {
			return fmt.Errorf("container '%s' uses forbidden image pattern '%s'", container.Name, forbidden)
		}
	}

	// Validate image format
	if err := v.validateImageFormat(container.Image); err != nil {
		return fmt.Errorf("container '%s' has invalid image: %w", container.Name, err)
	}

	// Validate resource requirements
	if err := v.validateContainerResources(container.Resources, container.Name); err != nil {
		return fmt.Errorf("container '%s' resource validation failed: %w", container.Name, err)
	}

	// Validate environment variables
	if err := v.validateEnvironmentVariables(container.Env, container.Name); err != nil {
		return fmt.Errorf("container '%s' environment validation failed: %w", container.Name, err)
	}

	// Validate ports
	if err := v.validateContainerPorts(container.Ports, container.Name); err != nil {
		return fmt.Errorf("container '%s' port validation failed: %w", container.Name, err)
	}

	return nil
}

// validateResourceRequirements validates resource requirements across the job
func (v *Validator) validateResourceRequirements(job *volcanov1alpha1.Job) error {
	totalCPU := resource.NewQuantity(0, resource.DecimalSI)
	totalMemory := resource.NewQuantity(0, resource.BinarySI)
	totalGPU := resource.NewQuantity(0, resource.DecimalSI)

	for _, task := range job.Spec.Tasks {
		for _, container := range task.Template.Spec.Containers {
			if container.Resources.Requests != nil {
				if cpu, ok := container.Resources.Requests[corev1.ResourceCPU]; ok {
					total := resource.NewMilliQuantity(0, resource.DecimalSI)
					total.Add(cpu)
					if task.Replicas > 1 {
						for i := int32(1); i < task.Replicas; i++ {
							total.Add(cpu)
						}
					}
					totalCPU.Add(*total)
				}

				if memory, ok := container.Resources.Requests[corev1.ResourceMemory]; ok {
					total := resource.NewQuantity(0, resource.BinarySI)
					total.Add(memory)
					if task.Replicas > 1 {
						for i := int32(1); i < task.Replicas; i++ {
							total.Add(memory)
						}
					}
					totalMemory.Add(*total)
				}

				if gpu, ok := container.Resources.Requests["nvidia.com/gpu"]; ok {
					total := resource.NewQuantity(0, resource.DecimalSI)
					total.Add(gpu)
					if task.Replicas > 1 {
						for i := int32(1); i < task.Replicas; i++ {
							total.Add(gpu)
						}
					}
					totalGPU.Add(*total)
				}
			}
		}
	}

	// Validate total resource consumption is reasonable
	if totalCPU.Cmp(*resource.NewQuantity(1000*1000, resource.DecimalSI)) > 0 { // > 1000 CPUs
		return fmt.Errorf("total CPU request (%s) exceeds reasonable limits", totalCPU.String())
	}

	if totalMemory.Cmp(*resource.NewQuantity(1024*1024*1024*1024, resource.BinarySI)) > 0 { // > 1TB
		return fmt.Errorf("total memory request (%s) exceeds reasonable limits", totalMemory.String())
	}

	return nil
}

// validateNetworking validates networking configuration
func (v *Validator) validateNetworking(job *volcanov1alpha1.Job) error {
	usedPorts := make(map[int32]string)

	for _, task := range job.Spec.Tasks {
		for _, container := range task.Template.Spec.Containers {
			for _, port := range container.Ports {
				if existingTask, exists := usedPorts[port.ContainerPort]; exists {
					return fmt.Errorf("port %d is used by both task '%s' and task '%s'", port.ContainerPort, existingTask, task.Name)
				}
				usedPorts[port.ContainerPort] = task.Name

				// Validate port range
				if port.ContainerPort < 1 || port.ContainerPort > 65535 {
					return fmt.Errorf("container port %d in task '%s' is out of valid range (1-65535)", port.ContainerPort, task.Name)
				}

				// Validate protocol
				if port.Protocol != "" && port.Protocol != corev1.ProtocolTCP && port.Protocol != corev1.ProtocolUDP {
					return fmt.Errorf("invalid protocol '%s' for port %d in task '%s'", port.Protocol, port.ContainerPort, task.Name)
				}
			}
		}
	}

	return nil
}

// validateSecurity validates security-related configuration
func (v *Validator) validateSecurity(job *volcanov1alpha1.Job) error {
	for _, task := range job.Spec.Tasks {
		podSpec := task.Template.Spec

		// Validate security context
		if podSpec.SecurityContext != nil {
			if err := v.validatePodSecurityContext(podSpec.SecurityContext, task.Name); err != nil {
				return fmt.Errorf("pod security context validation failed for task '%s': %w", task.Name, err)
			}
		}

		// Validate container security contexts
		for _, container := range podSpec.Containers {
			if container.SecurityContext != nil {
				if err := v.validateContainerSecurityContext(container.SecurityContext, container.Name, task.Name); err != nil {
					return fmt.Errorf("container security context validation failed: %w", err)
				}
			}
		}

		// Validate service account
		if podSpec.ServiceAccountName != "" {
			nameRegex := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
			if !nameRegex.MatchString(podSpec.ServiceAccountName) {
				return fmt.Errorf("invalid service account name '%s' in task '%s'", podSpec.ServiceAccountName, task.Name)
			}
		}
	}

	return nil
}

// CheckBestPractices checks for best practice violations
func (v *Validator) CheckBestPractices(job *volcanov1alpha1.Job) []string {
	warnings := make([]string, 0)

	// Check for resource limits
	for _, task := range job.Spec.Tasks {
		for _, container := range task.Template.Spec.Containers {
			if container.Resources.Limits == nil {
				warnings = append(warnings, fmt.Sprintf("Container '%s' in task '%s' has no resource limits", container.Name, task.Name))
			}

			if container.Resources.Requests == nil {
				warnings = append(warnings, fmt.Sprintf("Container '%s' in task '%s' has no resource requests", container.Name, task.Name))
			}
		}
	}

	// Check for health checks
	for _, task := range job.Spec.Tasks {
		for _, container := range task.Template.Spec.Containers {
			if container.LivenessProbe == nil {
				warnings = append(warnings, fmt.Sprintf("Container '%s' in task '%s' has no liveness probe", container.Name, task.Name))
			}

			if container.ReadinessProbe == nil {
				warnings = append(warnings, fmt.Sprintf("Container '%s' in task '%s' has no readiness probe", container.Name, task.Name))
			}
		}
	}

	// Check for security context
	for _, task := range job.Spec.Tasks {
		if task.Template.Spec.SecurityContext == nil {
			warnings = append(warnings, fmt.Sprintf("Task '%s' has no pod security context", task.Name))
		}

		for _, container := range task.Template.Spec.Containers {
			if container.SecurityContext == nil {
				warnings = append(warnings, fmt.Sprintf("Container '%s' in task '%s' has no security context", container.Name, task.Name))
			} else if container.SecurityContext.RunAsNonRoot == nil || !*container.SecurityContext.RunAsNonRoot {
				warnings = append(warnings, fmt.Sprintf("Container '%s' in task '%s' may run as root", container.Name, task.Name))
			}
		}
	}

	// Check for image tags
	for _, task := range job.Spec.Tasks {
		for _, container := range task.Template.Spec.Containers {
			if strings.HasSuffix(container.Image, ":latest") || !strings.Contains(container.Image, ":") {
				warnings = append(warnings, fmt.Sprintf("Container '%s' in task '%s' uses 'latest' tag or no tag", container.Name, task.Name))
			}
		}
	}

	return warnings
}

// Helper validation functions

func (v *Validator) validateLabelValue(key, value string) error {
	if len(value) > 63 {
		return fmt.Errorf("label value too long (max 63 characters)")
	}

	validRegex := regexp.MustCompile(`^[a-zA-Z0-9]([-_a-zA-Z0-9]*[a-zA-Z0-9])?$`)
	if value != "" && !validRegex.MatchString(value) {
		return fmt.Errorf("invalid label value format")
	}

	return nil
}

func (v *Validator) validateImageFormat(image string) error {
	// Basic image format validation
	if !strings.Contains(image, "/") && !strings.Contains(image, ":") {
		return fmt.Errorf("image format appears invalid: %s", image)
	}

	// Check for common typos
	if strings.Contains(image, " ") {
		return fmt.Errorf("image name contains spaces: %s", image)
	}

	return nil
}

func (v *Validator) validateContainerResources(resources corev1.ResourceRequirements, containerName string) error {
	// Validate resource requests
	if resources.Requests != nil {
		for resourceName, quantity := range resources.Requests {
			if quantity.IsZero() {
				return fmt.Errorf("resource request for '%s' cannot be zero", resourceName)
			}

			if quantity.Sign() < 0 {
				return fmt.Errorf("resource request for '%s' cannot be negative", resourceName)
			}
		}
	}

	// Validate resource limits
	if resources.Limits != nil {
		for resourceName, quantity := range resources.Limits {
			if quantity.Sign() < 0 {
				return fmt.Errorf("resource limit for '%s' cannot be negative", resourceName)
			}
		}
	}

	// Validate that limits >= requests
	if resources.Requests != nil && resources.Limits != nil {
		for resourceName, requestQuantity := range resources.Requests {
			if limitQuantity, ok := resources.Limits[resourceName]; ok {
				if limitQuantity.Cmp(requestQuantity) < 0 {
					return fmt.Errorf("resource limit for '%s' (%s) is less than request (%s)", resourceName, limitQuantity.String(), requestQuantity.String())
				}
			}
		}
	}

	return nil
}

func (v *Validator) validateEnvironmentVariables(envVars []corev1.EnvVar, containerName string) error {
	envNames := make(map[string]bool)

	for _, env := range envVars {
		if env.Name == "" {
			return fmt.Errorf("environment variable name cannot be empty")
		}

		if envNames[env.Name] {
			return fmt.Errorf("duplicate environment variable '%s'", env.Name)
		}
		envNames[env.Name] = true

		// Validate environment variable name format
		envNameRegex := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
		if !envNameRegex.MatchString(env.Name) {
			return fmt.Errorf("invalid environment variable name '%s'", env.Name)
		}
	}

	return nil
}

func (v *Validator) validateContainerPorts(ports []corev1.ContainerPort, containerName string) error {
	portNumbers := make(map[int32]bool)

	for _, port := range ports {
		if portNumbers[port.ContainerPort] {
			return fmt.Errorf("duplicate port %d in container '%s'", port.ContainerPort, containerName)
		}
		portNumbers[port.ContainerPort] = true

		if port.ContainerPort < 1 || port.ContainerPort > 65535 {
			return fmt.Errorf("port %d out of valid range (1-65535)", port.ContainerPort)
		}

		if port.Name != "" {
			nameRegex := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
			if !nameRegex.MatchString(port.Name) {
				return fmt.Errorf("invalid port name '%s'", port.Name)
			}
		}
	}

	return nil
}

func (v *Validator) validateVolumes(volumes []corev1.Volume, taskName string) error {
	volumeNames := make(map[string]bool)

	for _, volume := range volumes {
		if volume.Name == "" {
			return fmt.Errorf("volume name cannot be empty in task '%s'", taskName)
		}

		if volumeNames[volume.Name] {
			return fmt.Errorf("duplicate volume name '%s' in task '%s'", volume.Name, taskName)
		}
		volumeNames[volume.Name] = true

		nameRegex := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
		if !nameRegex.MatchString(volume.Name) {
			return fmt.Errorf("invalid volume name '%s' in task '%s'", volume.Name, taskName)
		}
	}

	return nil
}

func (v *Validator) validatePodSecurityContext(secCtx *corev1.PodSecurityContext, taskName string) error {
	if secCtx.RunAsUser != nil && *secCtx.RunAsUser < 0 {
		return fmt.Errorf("runAsUser cannot be negative in task '%s'", taskName)
	}

	if secCtx.RunAsGroup != nil && *secCtx.RunAsGroup < 0 {
		return fmt.Errorf("runAsGroup cannot be negative in task '%s'", taskName)
	}

	if secCtx.FSGroup != nil && *secCtx.FSGroup < 0 {
		return fmt.Errorf("fsGroup cannot be negative in task '%s'", taskName)
	}

	return nil
}

func (v *Validator) validateContainerSecurityContext(secCtx *corev1.SecurityContext, containerName, taskName string) error {
	if secCtx.RunAsUser != nil && *secCtx.RunAsUser < 0 {
		return fmt.Errorf("runAsUser cannot be negative for container '%s' in task '%s'", containerName, taskName)
	}

	if secCtx.RunAsGroup != nil && *secCtx.RunAsGroup < 0 {
		return fmt.Errorf("runAsGroup cannot be negative for container '%s' in task '%s'", containerName, taskName)
	}

	return nil
}

var nameRegex = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
