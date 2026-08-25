package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/conservation"
)

type selftestEnvelope struct {
	Data  json.RawMessage `json:"data"`
	Error *apiError       `json:"error"`
}

type selftestCaseResult struct {
	Case *conservation.ConservationCase `json:"case"`
}

func RunSelfTest(ctx context.Context, handler http.Handler) error {
	server := httptest.NewServer(handler)
	defer server.Close()
	client := &http.Client{Timeout: 3 * time.Second}
	if _, err := selftestRequest(ctx, client, http.MethodGet, server.URL+"/api/health", nil, http.StatusOK); err != nil {
		return fmt.Errorf("健康检查失败: %w", err)
	}
	evidence := []map[string]any{{"id": "ev-1", "filename": "damage-front.jpg", "media_type": "image/jpeg"}}
	create := map[string]any{
		"request_id": "self-create", "actor": map[string]any{"name": "自检修复师", "role": "conservator"},
		"id": "selftest-case", "shelf_mark": "善本-自检-001", "title": "自检古籍",
		"version_identifier": "卷一初刻本", "support_material": "竹纸",
		"carrier_characteristics": "线装，墨书，纸张薄韧", "damage_locations": []string{"书口", "下书角"},
		"initial_evidence": evidence, "responsible_conservator": "自检修复师",
	}
	item, err := selftestCase(ctx, client, http.MethodPost, server.URL+"/api/cases", create, http.StatusCreated)
	if err != nil {
		return err
	}
	item, err = selftestCase(ctx, client, http.MethodPost, server.URL+"/api/cases/selftest-case/submit", selftestMeta("self-submit", item.Revision, "自检修复师", "conservator"), http.StatusOK)
	if err != nil {
		return err
	}
	assessment := selftestMeta("self-assessment", item.Revision, "自检负责人", "manager")
	assessment["assessment"] = map[string]any{
		"severity": "moderate", "locations": []string{"书口", "下书角"}, "symptoms": []string{"撕裂", "脆化"},
		"probable_causes": []string{"机械磨损", "环境干燥"}, "treatment_goals": []string{"稳定纸张", "恢复翻阅功能"},
		"non_intervention_limits": []string{"保留原有污渍与题记"}, "evidence_refs": evidence, "assessor": "自检负责人",
		"partition_assessments": []map[string]any{
			{"location": "书口", "severity": "moderate", "symptoms": []string{"撕裂"}, "impact_scope": "书口局部", "treatment_goals": []string{"稳定纸张"}, "boundaries": []string{"保留题记"}},
			{"location": "下书角", "severity": "minor", "symptoms": []string{"脆化"}, "impact_scope": "下书角局部", "treatment_goals": []string{"恢复翻阅功能"}, "boundaries": []string{"保留污渍"}},
		},
	}
	item, err = selftestCase(ctx, client, http.MethodPost, server.URL+"/api/cases/selftest-case/assessment", assessment, http.StatusOK)
	if err != nil {
		return err
	}
	proposal := selftestMeta("self-proposal", item.Revision, "自检修复师", "conservator")
	proposal["proposal"] = map[string]any{
		"proposal_version":         "P1",
		"materials":                []map[string]any{{"name": "楮皮纸", "specification": "手工薄型", "purpose": "补纸"}},
		"procedure_steps":          []map[string]any{{"order": 1, "instruction": "低湿清洁后以楮皮纸补强", "checkpoint": "纤维搭接平整"}},
		"environment_requirements": []string{"温度 20±2℃，相对湿度 50±5%"},
		"reversibility_notes":      "补纸可经受控加湿移除", "risk_controls": []string{"先行小样，控制含水量"},
		"acceptance_criteria": []string{"补纸无翘曲", "墨迹无晕染"},
	}
	item, err = selftestCase(ctx, client, http.MethodPost, server.URL+"/api/cases/selftest-case/proposal", proposal, http.StatusOK)
	if err != nil {
		return err
	}
	for index, reviewer := range []string{"审议专家甲", "审议专家乙"} {
		review := selftestMeta(fmt.Sprintf("self-review-%d", index+1), item.Revision, reviewer, "reviewer")
		review["review"] = map[string]any{
			"id": fmt.Sprintf("review-%d", index+1), "proposal_version": "P1", "reviewer": reviewer, "decision": "approve",
		}
		item, err = selftestCase(ctx, client, http.MethodPost, server.URL+"/api/cases/selftest-case/reviews", review, http.StatusOK)
		if err != nil {
			return err
		}
	}
	trial := selftestMeta("self-trial", item.Revision, "自检修复师", "conservator")
	trial["trial"] = map[string]any{
		"id": "trial-1", "batch_code": "T-2026-001", "method": "同材质边角小样",
		"observations": []string{"干燥后平整", "墨色稳定"}, "deviations": []string{},
		"evidence_refs":     []map[string]any{{"id": "ev-trial", "filename": "trial.jpg", "media_type": "image/jpeg"}},
		"criterion_results": []map[string]any{{"criterion": "补纸无翘曲", "passed": true}, {"criterion": "墨迹无晕染", "passed": true}},
	}
	item, err = selftestCase(ctx, client, http.MethodPost, server.URL+"/api/cases/selftest-case/trial", trial, http.StatusOK)
	if err != nil {
		return err
	}
	archive := selftestMeta("self-archive", item.Revision, "自检负责人", "manager")
	archive["confirmed_hints"] = []string{"evidence.sha256"}
	item, err = selftestCase(ctx, client, http.MethodPost, server.URL+"/api/cases/selftest-case/archive", archive, http.StatusOK)
	if err != nil {
		return err
	}
	if item.Status != conservation.StatusArchived {
		return fmt.Errorf("最终状态为 %s，期望 archived", item.Status)
	}
	if _, err := selftestRequest(ctx, client, http.MethodGet, server.URL+"/api/cases/selftest-case/audit", nil, http.StatusOK); err != nil {
		return fmt.Errorf("审计包检查失败: %w", err)
	}
	return nil
}

func selftestMeta(requestID string, revision int64, name, role string) map[string]any {
	return map[string]any{"request_id": requestID, "expected_revision": revision, "actor": map[string]any{"name": name, "role": role}}
}

func selftestCase(ctx context.Context, client *http.Client, method, url string, body any, expected int) (*conservation.ConservationCase, error) {
	payload, err := selftestRequest(ctx, client, method, url, body, expected)
	if err != nil {
		return nil, err
	}
	var result selftestCaseResult
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, fmt.Errorf("解析事项响应: %w", err)
	}
	if result.Case == nil {
		return nil, fmt.Errorf("事项响应缺少 case")
	}
	return result.Case, nil
}

func selftestRequest(ctx context.Context, client *http.Client, method, url string, body any, expected int) (json.RawMessage, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(data)
	}
	request, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	var wrapped selftestEnvelope
	if err := json.Unmarshal(data, &wrapped); err != nil {
		return nil, fmt.Errorf("响应不是 JSON: %w", err)
	}
	if response.StatusCode != expected {
		return nil, fmt.Errorf("%s %s 返回 %d，期望 %d：%s", method, url, response.StatusCode, expected, string(data))
	}
	if wrapped.Error != nil {
		return nil, fmt.Errorf("API 返回错误 %s: %s", wrapped.Error.Code, wrapped.Error.Message)
	}
	return wrapped.Data, nil
}
