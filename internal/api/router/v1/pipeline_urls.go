package v1

import (
	"github.com/gin-gonic/gin"
	pipelinehandler "github.com/plutolove233/co-dream/internal/api/handler/pipeline"
	"github.com/plutolove233/co-dream/internal/api/middleware"
)

func RegisterPipelineRouters(engine *gin.RouterGroup) {
	api := pipelinehandler.NewAPI()
	group := engine.Group("")
	group.Use(middleware.TokenRequired())
	{
		group.POST("/pipelines", api.CreatePipeline)
		group.GET("/pipelines", api.ListPipelines)
		group.GET("/pipelines/:id", api.GetPipeline)
		group.PUT("/pipelines/:id", api.UpdatePipeline)
		group.DELETE("/pipelines/:id", api.DeletePipeline)
		group.POST("/pipelines/:id/execute", api.ExecutePipeline)

		group.GET("/executions/:id", api.GetExecution)
		group.GET("/executions/:id/events", api.StreamExecutionEvents)
		group.GET("/executions/:id/stages", api.ListExecutionStages)
		group.PATCH("/executions/:id", api.UpdateExecutionAction)

		group.GET("/checkpoints", api.ListPendingCheckpoints)
		group.POST("/checkpoints/:id/approve", api.ApproveCheckpoint)
		group.POST("/checkpoints/:id/reject", api.RejectCheckpoint)
	}
}
