package model

import "errors"

var (
	ErrResumeBuilderNotFound = errors.New("resume builder not found")
	ErrNotOwner              = errors.New("not the owner of this resume")
	ErrSectionEntryNotFound  = errors.New("section entry not found")
	ErrInvalidTemplate       = errors.New("invalid template")
	ErrInvalidSpacing        = errors.New("spacing must be between 50 and 150")
	ErrInvalidColor          = errors.New("invalid color format")
	ErrInvalidFont           = errors.New("invalid font family")
	ErrInvalidSectionKey     = errors.New("invalid section key")
	ErrInvalidMargin         = errors.New("margin must be between 0 and 200")
	ErrInvalidLayoutMode     = errors.New("layout mode must be single, double-left, double-right, or custom")
	ErrInvalidSidebarWidth   = errors.New("sidebar width must be between 25 and 50")
	ErrInvalidColumnValue    = errors.New("column must be main or sidebar")
	ErrInvalidFontSize       = errors.New("font size must be between 8 and 18")
	ErrInvalidSkillDisplay   = errors.New("invalid skill display mode")
)

// ErrorCode represents error codes for the resume builder module.
type ErrorCode string

const (
	CodeResumeBuilderNotFound ErrorCode = "RESUME_BUILDER_NOT_FOUND"
	CodeNotOwner              ErrorCode = "NOT_OWNER"
	CodeSectionEntryNotFound  ErrorCode = "SECTION_ENTRY_NOT_FOUND"
	CodeInvalidTemplate       ErrorCode = "INVALID_TEMPLATE"
	CodeInvalidSpacing        ErrorCode = "INVALID_SPACING"
	CodeInvalidColor          ErrorCode = "INVALID_COLOR"
	CodeInvalidFont           ErrorCode = "INVALID_FONT"
	CodeInvalidSectionKey     ErrorCode = "INVALID_SECTION_KEY"
	CodeInvalidMargin         ErrorCode = "INVALID_MARGIN"
	CodeInvalidLayoutMode     ErrorCode = "INVALID_LAYOUT_MODE"
	CodeInvalidSidebarWidth   ErrorCode = "INVALID_SIDEBAR_WIDTH"
	CodeInvalidColumnValue    ErrorCode = "INVALID_COLUMN_VALUE"
	CodeInvalidFontSize       ErrorCode = "INVALID_FONT_SIZE"
	CodeInvalidSkillDisplay   ErrorCode = "INVALID_SKILL_DISPLAY"
	CodeInternalError         ErrorCode = "INTERNAL_ERROR"
)

// GetErrorCode maps errors to error codes.
func GetErrorCode(err error) ErrorCode {
	switch {
	case errors.Is(err, ErrResumeBuilderNotFound):
		return CodeResumeBuilderNotFound
	case errors.Is(err, ErrNotOwner):
		return CodeNotOwner
	case errors.Is(err, ErrSectionEntryNotFound):
		return CodeSectionEntryNotFound
	case errors.Is(err, ErrInvalidTemplate):
		return CodeInvalidTemplate
	case errors.Is(err, ErrInvalidSpacing):
		return CodeInvalidSpacing
	case errors.Is(err, ErrInvalidColor):
		return CodeInvalidColor
	case errors.Is(err, ErrInvalidFont):
		return CodeInvalidFont
	case errors.Is(err, ErrInvalidSectionKey):
		return CodeInvalidSectionKey
	case errors.Is(err, ErrInvalidMargin):
		return CodeInvalidMargin
	case errors.Is(err, ErrInvalidLayoutMode):
		return CodeInvalidLayoutMode
	case errors.Is(err, ErrInvalidSidebarWidth):
		return CodeInvalidSidebarWidth
	case errors.Is(err, ErrInvalidColumnValue):
		return CodeInvalidColumnValue
	case errors.Is(err, ErrInvalidFontSize):
		return CodeInvalidFontSize
	case errors.Is(err, ErrInvalidSkillDisplay):
		return CodeInvalidSkillDisplay
	default:
		return CodeInternalError
	}
}

// GetErrorMessage returns a user-friendly error message.
func GetErrorMessage(err error) string {
	switch {
	case errors.Is(err, ErrResumeBuilderNotFound):
		return "Resume builder not found"
	case errors.Is(err, ErrNotOwner):
		return "You don't have access to this resume"
	case errors.Is(err, ErrSectionEntryNotFound):
		return "Section entry not found"
	case errors.Is(err, ErrInvalidTemplate):
		return "Invalid template selected"
	case errors.Is(err, ErrInvalidSpacing):
		return "Spacing must be between 50 and 150"
	case errors.Is(err, ErrInvalidColor):
		return "Invalid color format"
	case errors.Is(err, ErrInvalidFont):
		return "Invalid font family"
	case errors.Is(err, ErrInvalidSectionKey):
		return "Invalid section key"
	case errors.Is(err, ErrInvalidMargin):
		return "Margin must be between 0 and 200"
	case errors.Is(err, ErrInvalidLayoutMode):
		return "Layout mode must be single, double-left, double-right, or custom"
	case errors.Is(err, ErrInvalidSidebarWidth):
		return "Sidebar width must be between 25 and 50"
	case errors.Is(err, ErrInvalidColumnValue):
		return "Column must be main or sidebar"
	case errors.Is(err, ErrInvalidFontSize):
		return "Font size must be between 8 and 18"
	case errors.Is(err, ErrInvalidSkillDisplay):
		return "Invalid skill display mode"
	default:
		return "Internal server error"
	}
}
