package transcript

import (
	"strings"
	"time"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func (s *TranscriptService) runWorker() {
	// Jobs that could not join the current merge window; FIFO for later.
	buffered := make([]transcriptJob, 0, cap(s.queue))

	for {
		primary := func() transcriptJob {
			if len(buffered) > 0 {
				j := buffered[0]
				buffered = buffered[1:]
				return j
			}
			return <-s.queue
		}()

		baseActiveFile := strings.TrimSpace(primary.params.ActiveFile)

		// Coalesce group: several RPCs merged into one Execute (not a directive-apply batch).
		coalesceGroup := []transcriptJob{primary}
		mergedTextParts := []string{primary.params.Text}
		mergedChars := len(primary.params.Text)

		timer := time.NewTimer(s.coalesceWindow)

		for collecting := true; collecting; {
			select {
			case j := <-s.queue:
				activeFile := strings.TrimSpace(j.params.ActiveFile)
				text := strings.TrimSpace(j.params.Text)
				if text == "" {
					j.resp <- transcriptAcceptResp{
						result: protocol.VoiceTranscriptResult{Success: true},
						ok:     true,
					}
					continue
				}

				eligible := activeFile == baseActiveFile &&
					len(coalesceGroup) < s.maxMergeJobs &&
					mergedChars+1+len(text) <= s.maxMergeChars

				if eligible {
					j.params.Text = text
					coalesceGroup = append(coalesceGroup, j)
					mergedTextParts = append(mergedTextParts, text)
					mergedChars += 1 + len(text)
				} else {
					buffered = append(buffered, j)
				}
			case <-timer.C:
				collecting = false
			}
		}

		timer.Stop()

		mergedParams := primary.params
		mergedParams.Text = strings.Join(mergedTextParts, " ")

		mergedResult, ok := s.runExecute(mergedParams)

		// Primary job carries the real result; coalesced jobs no-op success to avoid duplicate applies.
		for i, j := range coalesceGroup {
			if i == 0 {
				j.resp <- transcriptAcceptResp{
					result: mergedResult,
					ok:     ok,
				}
			} else {
				j.resp <- transcriptAcceptResp{
					result: protocol.VoiceTranscriptResult{Success: true},
					ok:     true,
				}
			}
		}
	}
}
