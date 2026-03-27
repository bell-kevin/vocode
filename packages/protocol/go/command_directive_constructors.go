package protocol

func NewCommandDirective(command string, args []string, timeoutMs *int64) CommandDirective {
	d := CommandDirective{
		Command: command,
		Args:    args,
	}
	if timeoutMs != nil {
		d.TimeoutMs = timeoutMs
	}
	return d
}
