// Package ui provides output abstractions for TPM.
package ui

// Output abstracts message display for different output modes.
type Output interface {
	// Ok displays a success/informational message.
	Ok(msg string)

	// Err displays an error message and marks the output as failed.
	Err(msg string)

	// EndMessage displays the completion message with instructions.
	EndMessage()

	// HasFailed returns true if any Err calls were made.
	HasFailed() bool
}
