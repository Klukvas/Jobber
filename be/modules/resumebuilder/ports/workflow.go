package ports

import (
	"context"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
)

// WorkflowTransaction shares the commit with resume creation and the Job link.
type WorkflowTransaction interface {
	TailoringTransaction
	GetOperation(context.Context, string, string, string) (*model.WorkflowOperation, error)
	PutOperation(context.Context, *model.WorkflowOperation) error
	FindImport(context.Context, string, string, string) (*model.WorkflowOperation, error)
	ChargeAI(context.Context, string, string, int) error
}
