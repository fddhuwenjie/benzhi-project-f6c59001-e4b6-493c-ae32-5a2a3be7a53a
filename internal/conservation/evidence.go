package conservation

import (
	"strings"
	"time"
)

type EvidenceRef struct {
	ID         string    `json:"id"`
	Filename   string    `json:"filename"`
	MediaType  string    `json:"media_type"`
	SHA256     string    `json:"sha256,omitempty"`
	CapturedAt time.Time `json:"captured_at"`
	Note       string    `json:"note,omitempty"`
}

func validateEvidence(field string, refs []EvidenceRef, required bool, v *ValidationError) {
	if required && len(refs) == 0 {
		v.Add(field, "至少需要一项图像或文档证据")
	}
	seen := make(map[string]struct{}, len(refs))
	for i, ref := range refs {
		prefix := field + "[" + itoa(i) + "]"
		ref.ID = strings.TrimSpace(ref.ID)
		ref.Filename = strings.TrimSpace(ref.Filename)
		ref.MediaType = strings.TrimSpace(ref.MediaType)
		if ref.ID == "" {
			v.Add(prefix+".id", "证据标识不能为空")
		}
		if strings.TrimSpace(ref.Filename) == "" {
			v.Add(prefix+".filename", "文件名不能为空")
		}
		if strings.TrimSpace(ref.MediaType) == "" {
			v.Add(prefix+".media_type", "媒体类型不能为空")
		}
		if _, ok := seen[ref.ID]; ok && ref.ID != "" {
			v.Add(prefix+".id", "证据标识不能重复")
		}
		seen[ref.ID] = struct{}{}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
