package openai

import (
	adaptercommon "chat/adapter/common"
	"chat/globals"
	"chat/utils"
	"fmt"
	"time"
)

type VideoRequest struct {
	Prompt         string  `json:"prompt"`
	Model          string  `json:"model,omitempty"`
	Seconds        *string `json:"seconds,omitempty"`
	Size           *string `json:"size,omitempty"`
	InputReference *string `json:"input_reference,omitempty"`
}

type VideoJob struct {
	CompletedAt        *int64  `json:"completed_at,omitempty"`
	CreatedAt          int64   `json:"created_at"`
	ExpiresAt          *int64  `json:"expires_at,omitempty"`
	Id                 string  `json:"id"`
	Model              string  `json:"model"`
	Object             string  `json:"object"`
	Progress           *int    `json:"progress,omitempty"`
	Prompt             string  `json:"prompt"`
	RemixedFromVideoId *string `json:"remixed_from_video_id,omitempty"`
	Seconds            string  `json:"seconds"`
	Size               string  `json:"size"`
	Status             string  `json:"status"`
	Error              *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *ChatInstance) getVideoCreateEndpoint() string {
	return c.GetAPIEndpoint("videos")
}

func (c *ChatInstance) getVideoQueryEndpoint(id string) string {
	return c.GetAPIEndpoint(fmt.Sprintf("videos/%s", id))
}

func (c *ChatInstance) CreateVideoRequest(props *adaptercommon.VideoProps, hook globals.Hook) error {
	body := VideoRequest{
		Prompt:         props.Prompt,
		Model:          props.Model,
		Seconds:        props.Seconds,
		Size:           props.Size,
		InputReference: props.InputReference,
	}

	res, err := utils.Post(c.getVideoCreateEndpoint(), c.GetHeader(), body, props.Proxy)
	if err != nil || res == nil {
		if err != nil {
			return fmt.Errorf("openai video error: %s", err.Error())
		}
		return fmt.Errorf("openai video error: empty response")
	}

	job := utils.MapToStruct[VideoJob](res)
	if job == nil {
		return fmt.Errorf("openai video error: cannot parse response")
	}
	if job.Error != nil && (job.Error.Message != "") {
		return fmt.Errorf("openai video error: %s", job.Error.Message)
	}

	const maxTimeout = 30 * time.Minute
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	deadline := time.After(maxTimeout)

	var begin bool
	var lastProgress int = -1

	for {
		if job.Status == "completed" {
			if begin {
				if err := hook(&globals.Chunk{Content: "```\n"}); err != nil {
					return err
				}
			}
			return hook(&globals.Chunk{Content: utils.Marshal(job)})
		}
		if job.Status == "failed" {
			if begin {
				if err := hook(&globals.Chunk{Content: "```\n"}); err != nil {
					return err
				}
			}
			if job.Error != nil && job.Error.Message != "" {
				return fmt.Errorf("openai video job failed: %s", job.Error.Message)
			}
			return fmt.Errorf("openai video job failed")
		}

		select {
		case <-ticker.C:
			if job.Id == "" {
				return hook(&globals.Chunk{Content: utils.Marshal(job)})
			}
			data, gErr := utils.Get(c.getVideoQueryEndpoint(job.Id), c.GetHeader(), props.Proxy)
			if gErr != nil || data == nil {
				continue
			}
			if j := utils.MapToStruct[VideoJob](data); j != nil {
				job = j
			}

			progress := 0
			if job.Progress != nil {
				progress = *job.Progress
			}

			if !begin {
				begin = true
				if err := hook(&globals.Chunk{Content: "```progress\n"}); err != nil {
					return err
				}
			}

			if progress != lastProgress {
				if err := hook(&globals.Chunk{Content: fmt.Sprintf("%d\n", progress)}); err != nil {
					return err
				}
				lastProgress = progress
			}
		case <-deadline:
			if begin {
				if err := hook(&globals.Chunk{Content: "```\n"}); err != nil {
					return err
				}
			}
			return fmt.Errorf("openai video job timeout")
		}
	}
}
