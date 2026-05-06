package pipeline

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/plutolove233/co-dream/internal/api/service"
	"github.com/plutolove233/co-dream/internal/globals"
)

type API struct {
	engine *service.PipelineEngineService
}

func NewAPI() *API {
	return &API{engine: service.DefaultPipelineEngine()}
}

func (a *API) CreatePipeline(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req service.PipelineCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		globals.JsonParameterIllegal(c, "请求参数不符合要求", err)
		return
	}
	record, err := a.engine.CreatePipeline(c.Request.Context(), userID, req)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	globals.JsonOK(c, "创建 Pipeline 成功", record)
}

func (a *API) ListPipelines(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	globals.JsonOK(c, "查询 Pipeline 列表成功", a.engine.ListPipelines(c.Request.Context(), userID))
}

func (a *API) GetPipeline(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	record, err := a.engine.GetPipeline(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	globals.JsonOK(c, "查询 Pipeline 成功", record)
}

func (a *API) UpdatePipeline(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req service.PipelineUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		globals.JsonParameterIllegal(c, "请求参数不符合要求", err)
		return
	}
	record, err := a.engine.UpdatePipeline(c.Request.Context(), userID, c.Param("id"), req)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	globals.JsonOK(c, "更新 Pipeline 成功", record)
}

func (a *API) DeletePipeline(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	if err := a.engine.DeletePipeline(c.Request.Context(), userID, c.Param("id")); err != nil {
		writeServiceError(c, err)
		return
	}
	globals.JsonOK(c, "删除 Pipeline 成功", nil)
}

func (a *API) ExecutePipeline(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req service.PipelineExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		globals.JsonParameterIllegal(c, "请求参数不符合要求", err)
		return
	}
	record, err := a.engine.StartExecution(c.Request.Context(), userID, c.Param("id"), req)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	globals.JsonOK(c, "启动 Pipeline 执行成功", record)
}

func (a *API) GetExecution(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	record, err := a.engine.GetExecution(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	globals.JsonOK(c, "查询执行状态成功", record)
}

func (a *API) StreamExecutionEvents(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	events, unsubscribe, err := a.engine.SubscribeExecution(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	defer unsubscribe()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	c.Stream(func(w io.Writer) bool {
		select {
		case <-c.Request.Context().Done():
			return false
		case event, ok := <-events:
			if !ok {
				return false
			}
			c.SSEvent(event.Type, event)
			return true
		}
	})
}

func (a *API) ListExecutionStages(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	stages, err := a.engine.ListExecutionStages(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	globals.JsonOK(c, "查询执行阶段成功", stages)
}

func (a *API) UpdateExecutionAction(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req service.ExecutionActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		globals.JsonParameterIllegal(c, "请求参数不符合要求", err)
		return
	}
	record, err := a.engine.UpdateExecutionAction(c.Request.Context(), userID, c.Param("id"), req)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	globals.JsonOK(c, "更新执行状态成功", record)
}

func (a *API) ListPendingCheckpoints(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	globals.JsonOK(c, "查询待审批检查点成功", a.engine.ListPendingCheckpoints(c.Request.Context(), userID))
}

func (a *API) ApproveCheckpoint(c *gin.Context) {
	a.decideCheckpoint(c, true)
}

func (a *API) RejectCheckpoint(c *gin.Context) {
	a.decideCheckpoint(c, false)
}

func (a *API) decideCheckpoint(c *gin.Context, approve bool) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req service.CheckpointDecisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		globals.JsonParameterIllegal(c, "请求参数不符合要求", err)
		return
	}

	var (
		record service.ExecutionRecord
		err    error
	)
	if approve {
		record, err = a.engine.ApproveCheckpoint(c.Request.Context(), userID, c.Param("id"), req)
	} else {
		record, err = a.engine.RejectCheckpoint(c.Request.Context(), userID, c.Param("id"), req)
	}
	if err != nil {
		writeServiceError(c, err)
		return
	}
	globals.JsonOK(c, "处理检查点成功", record)
}

func currentUserID(c *gin.Context) (string, bool) {
	v, ok := c.Get("id")
	if !ok {
		globals.JsonAccessDenied(c, "未登录")
		return "", false
	}
	userID, ok := v.(string)
	if !ok || userID == "" {
		globals.JsonAccessDenied(c, "用户信息无效")
		return "", false
	}
	return userID, true
}

func writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrPipelineNotFound),
		errors.Is(err, service.ErrExecutionNotFound),
		errors.Is(err, service.ErrCheckpointNotFound):
		globals.JsonDataError(c, "数据不存在", err)
	case errors.Is(err, service.ErrExecutionTerminal),
		errors.Is(err, service.ErrExecutionNotPaused):
		globals.JsonParameterIllegal(c, "执行状态不允许该操作", err)
	default:
		globals.JsonParameterIllegal(c, "Pipeline 参数不符合要求", err)
	}
}
