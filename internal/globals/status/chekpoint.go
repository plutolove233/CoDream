package status

const (
	CheckpointStatusPending  string = "pending"
	CheckpointStatusApproved string = "approved"
	CheckpointStatusRejected string = "rejected"
)

const (
	CheckpointBefore string = "before"
	CheckpointAfter  string = "after"
)

const (
	BackoffExponential string = "exponential"
	BackoffLinear      string = "linear"
	BackoffFixed       string = "fixed"
)
