package tools

import "github.com/mark3labs/mcp-go/mcp"

// readOnlyAnnotation is for tools that don't change cluster state.
// Repeated calls have no additional effect (idempotent). The cluster is an
// external system, so openWorld is true.
func readOnlyAnnotation(title string) mcp.ToolOption {
	return mcp.WithToolAnnotation(mcp.ToolAnnotation{
		Title:           title,
		ReadOnlyHint:    mcp.ToBoolPtr(true),
		DestructiveHint: mcp.ToBoolPtr(false),
		IdempotentHint:  mcp.ToBoolPtr(true),
		OpenWorldHint:   mcp.ToBoolPtr(true),
	})
}

// destructiveAnnotation is for tools that delete or otherwise destroy
// cluster state. Not idempotent — a second delete typically returns
// "not found".
func destructiveAnnotation(title string) mcp.ToolOption {
	return mcp.WithToolAnnotation(mcp.ToolAnnotation{
		Title:           title,
		ReadOnlyHint:    mcp.ToBoolPtr(false),
		DestructiveHint: mcp.ToBoolPtr(true),
		IdempotentHint:  mcp.ToBoolPtr(false),
		OpenWorldHint:   mcp.ToBoolPtr(true),
	})
}

// creationAnnotation is for tools that create a new resource. Not
// destructive in the data-loss sense, but not idempotent — recreating a
// resource that already exists errors with "already exists".
func creationAnnotation(title string) mcp.ToolOption {
	return mcp.WithToolAnnotation(mcp.ToolAnnotation{
		Title:           title,
		ReadOnlyHint:    mcp.ToBoolPtr(false),
		DestructiveHint: mcp.ToBoolPtr(false),
		IdempotentHint:  mcp.ToBoolPtr(false),
		OpenWorldHint:   mcp.ToBoolPtr(true),
	})
}

// idempotentMutationAnnotation is for update / scale / suspend / resume
// tools that drive a resource toward a desired final state. Repeating
// with the same arguments produces no further change.
func idempotentMutationAnnotation(title string) mcp.ToolOption {
	return mcp.WithToolAnnotation(mcp.ToolAnnotation{
		Title:           title,
		ReadOnlyHint:    mcp.ToBoolPtr(false),
		DestructiveHint: mcp.ToBoolPtr(false),
		IdempotentHint:  mcp.ToBoolPtr(true),
		OpenWorldHint:   mcp.ToBoolPtr(true),
	})
}
