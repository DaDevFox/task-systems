package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"

	"github.com/DaDevFox/task-systems/tasker-core/backend/internal/domain"
)

func (s *TaskService) validateTaskCompletionReadiness(ctx context.Context, task *domain.Task) error {
	if task == nil {
		return fmt.Errorf("task is required")
	}

	err := s.validateTaskResults(task)
	if err != nil {
		return err
	}

	return s.validateTaskResources(ctx, task)
}

func (s *TaskService) validateTaskResults(task *domain.Task) error {
	for _, result := range task.Results {
		if !result.Complete {
			return fmt.Errorf("result requirement %s is not complete", result.RequirementID)
		}
		if result.Result.FileLocationPath == "" {
			continue
		}
		_, err := os.Stat(result.Result.FileLocationPath)
		if err == nil {
			continue
		}
		return fmt.Errorf("result file does not exist for %s: %w", result.RequirementID, err)
	}

	return nil
}

func (s *TaskService) validateTaskResources(ctx context.Context, task *domain.Task) error {
	for _, resource := range task.Resources {
		err := s.validateResourceDependency(ctx, resource)
		if err == nil {
			continue
		}
		return err
	}

	return nil
}

func (s *TaskService) validateResourceDependency(ctx context.Context, resource domain.ResourceDependency) error {
	if resource.APIURL != "" {
		return s.validateAPIResource(ctx, resource.APIURL, resource.APIResponseRegex)
	}
	if resource.FileLocationPath != "" {
		_, err := os.Stat(resource.FileLocationPath)
		if err != nil {
			return fmt.Errorf("resource file dependency missing: %w", err)
		}
		return nil
	}

	return nil
}

func (s *TaskService) validateAPIResource(ctx context.Context, apiURL, responseRegex string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return fmt.Errorf("build api resource request failed: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("api resource request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("api resource responded with status %d", resp.StatusCode)
	}
	if responseRegex == "" {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read api resource response failed: %w", err)
	}

	matches, err := regexp.MatchString(responseRegex, string(body))
	if err != nil {
		return fmt.Errorf("invalid api response regex: %w", err)
	}
	if matches {
		return nil
	}

	return fmt.Errorf("api resource response did not match required regex")
}
