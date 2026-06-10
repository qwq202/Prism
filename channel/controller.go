package channel

import (
	"chat/utils"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/mail"
	"strings"
)

type SyncChargeForm struct {
	Overwrite bool           `json:"overwrite"`
	Data      ChargeSequence `json:"data"`
}

type TestMailForm struct {
	Email string `json:"email" binding:"required"`
}

func GetInfo(c *gin.Context) {
	c.JSON(http.StatusOK, SystemInstance.AsInfo())
}

func AttachmentService(c *gin.Context) {
	hash := c.Param("hash")
	utils.ServeStoredAttachment(c, hash)
}

func DeleteChannel(c *gin.Context) {
	id := c.Param("id")
	state := ConduitInstance.DeleteChannel(utils.ParseInt(id))
	if state == nil {
		ConduitInstance.Load()
	}

	c.JSON(http.StatusOK, gin.H{
		"status": state == nil,
		"error":  utils.GetError(state),
	})
}

func ActivateChannel(c *gin.Context) {
	id := c.Param("id")
	state := ConduitInstance.ActivateChannel(utils.ParseInt(id))
	if state == nil {
		ConduitInstance.Load()
	}

	c.JSON(http.StatusOK, gin.H{
		"status": state == nil,
		"error":  utils.GetError(state),
	})
}

func DeactivateChannel(c *gin.Context) {
	id := c.Param("id")
	state := ConduitInstance.DeactivateChannel(utils.ParseInt(id))
	if state == nil {
		ConduitInstance.Load()
	}

	c.JSON(http.StatusOK, gin.H{
		"status": state == nil,
		"error":  utils.GetError(state),
	})
}

func GetChannelList(c *gin.Context) {
	data := ConduitInstance.Sequence
	if data == nil {
		data = Sequence{}
	}

	c.JSON(http.StatusOK, gin.H{
		"status": true,
		"data":   data,
	})
}

func GetChannel(c *gin.Context) {
	id := c.Param("id")
	channel := ConduitInstance.Sequence.GetChannelById(utils.ParseInt(id))

	c.JSON(http.StatusOK, gin.H{
		"status": channel != nil,
		"data":   channel,
	})
}

func CreateChannel(c *gin.Context) {
	var channel Channel
	if err := c.ShouldBindJSON(&channel); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": false,
			"error":  err.Error(),
		})
		return
	}

	state := ConduitInstance.CreateChannel(&channel)
	if state == nil {
		ConduitInstance.Load()
	}
	c.JSON(http.StatusOK, gin.H{
		"status": state == nil,
		"error":  utils.GetError(state),
	})
}

func UpdateChannel(c *gin.Context) {
	var channel Channel
	if err := c.ShouldBindJSON(&channel); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": false,
			"error":  err.Error(),
		})
		return
	}

	id := c.Param("id")
	channel.Id = utils.ParseInt(id)

	state := ConduitInstance.UpdateChannel(channel.Id, &channel)
	if state == nil {
		ConduitInstance.Load()
	}
	c.JSON(http.StatusOK, gin.H{
		"status": state == nil,
		"error":  utils.GetError(state),
	})
}

func SetCharge(c *gin.Context) {
	var charge Charge
	if err := c.ShouldBindJSON(&charge); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": false,
			"error":  err.Error(),
		})
		return
	}

	state := ChargeInstance.SetRule(charge)
	c.JSON(http.StatusOK, gin.H{
		"status": state == nil,
		"error":  utils.GetError(state),
	})
}

func GetChargeList(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": true,
		"data":   ChargeInstance.ListRules(),
	})
}

func DeleteCharge(c *gin.Context) {
	id := c.Param("id")
	state := ChargeInstance.DeleteRule(utils.ParseInt(id))

	c.JSON(http.StatusOK, gin.H{
		"status": state == nil,
		"error":  utils.GetError(state),
	})
}

func SyncCharge(c *gin.Context) {
	var form SyncChargeForm
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": false,
			"error":  err.Error(),
		})
	}

	state := ChargeInstance.SyncRules(form.Data, form.Overwrite)
	c.JSON(http.StatusOK, gin.H{
		"status": state == nil,
		"error":  utils.GetError(state),
	})
}

func GetConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": true,
		"data":   SystemInstance,
	})
}

func UpdateConfig(c *gin.Context) {
	var config SystemConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": false,
			"error":  err.Error(),
		})
		return
	}

	state := SystemInstance.UpdateConfig(&config)
	c.JSON(http.StatusOK, gin.H{
		"status": state == nil,
		"error":  utils.GetError(state),
	})
}

func TestStorageConfig(c *gin.Context) {
	var config SystemConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": false,
			"error":  err.Error(),
		})
		return
	}

	err := utils.TestStorageConnection(utils.StorageTestConfig{
		Mode:    config.GetStorageMode(),
		Backend: config.GetBackend(),
		S3: utils.StorageS3Config{
			Endpoint:       config.GetStorageS3Endpoint(),
			Region:         config.GetStorageS3Region(),
			Bucket:         config.GetStorageS3Bucket(),
			AccessKey:      config.GetStorageS3AccessKey(),
			SecretKey:      config.GetStorageS3SecretKey(),
			PublicBaseURL:  config.GetStorageS3PublicBaseURL(),
			ForcePathStyle: config.Common.S3.ForcePathStyle,
		},
		R2: utils.StorageR2Config{
			AccountID:     config.GetStorageR2AccountID(),
			Jurisdiction:  config.GetStorageR2Jurisdiction(),
			Bucket:        config.GetStorageR2Bucket(),
			AccessKey:     config.GetStorageR2AccessKey(),
			SecretKey:     config.GetStorageR2SecretKey(),
			PublicBaseURL: config.GetStorageR2PublicBaseURL(),
		},
	})

	response := gin.H{
		"status": err == nil,
		"error":  utils.GetError(err),
	}
	if err == nil {
		response["message"] = "storage test passed"
	}
	c.JSON(http.StatusOK, response)
}

func TestMailConfig(c *gin.Context) {
	var form TestMailForm
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": false,
			"error":  err.Error(),
		})
		return
	}

	address, err := mail.ParseAddress(strings.TrimSpace(form.Email))
	if err != nil || address.Address == "" {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  "invalid email format",
		})
		return
	}

	err = SystemInstance.SendTestMail(address.Address)
	response := gin.H{
		"status": err == nil,
		"error":  utils.GetError(err),
	}
	if err == nil {
		response["message"] = "mail test passed"
	}
	c.JSON(http.StatusOK, response)
}

func GetPlanConfig(c *gin.Context) {
	c.JSON(http.StatusOK, PlanInstance)
}

func UpdatePlanConfig(c *gin.Context) {
	var config PlanManager
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": false,
			"error":  err.Error(),
		})
		return
	}

	state := PlanInstance.UpdateConfig(&config)
	c.JSON(http.StatusOK, gin.H{
		"status": state == nil,
		"error":  utils.GetError(state),
	})
}
