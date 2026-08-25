package conservation

import (
	"strings"
	"time"
)

type Severity string

const (
	SeverityMinor    Severity = "minor"
	SeverityModerate Severity = "moderate"
	SeveritySevere   Severity = "severe"
)

type DamageAssessment struct {
	CaseID                   string                     `json:"case_id"`
	Severity                 Severity                   `json:"severity"`
	Locations                []string                   `json:"locations"`
	Symptoms                 []string                   `json:"symptoms"`
	ProbableCauses           []string                   `json:"probable_causes"`
	TreatmentGoals           []string                   `json:"treatment_goals"`
	NonInterventionLimits    []string                   `json:"non_intervention_limits"`
	EvidenceRefs             []EvidenceRef              `json:"evidence_refs"`
	Assessor                 string                     `json:"assessor"`
	PartitionAssessments     []DamageLocationAssessment `json:"partition_assessments,omitempty"`
	LocationAssessments      []DamageLocationAssessment `json:"location_assessments,omitempty"`
	HighestPartitionSeverity Severity                   `json:"highest_partition_severity,omitempty"`
	ConfirmedAt              time.Time                  `json:"confirmed_at"`
}

type DamageLocationAssessment struct {
	Location              string   `json:"location"`
	Severity              Severity `json:"severity"`
	Symptoms              []string `json:"symptoms"`
	ImpactScope           string   `json:"impact_scope"`
	TreatmentGoals        []string `json:"treatment_goals"`
	Boundaries            []string `json:"boundaries"`
	Goals                 []string `json:"goals,omitempty"`
	NonInterventionLimits []string `json:"non_intervention_limits,omitempty"`
}

func severityRank(s Severity) int {
	switch s {
	case SeveritySevere:
		return 3
	case SeverityModerate:
		return 2
	case SeverityMinor:
		return 1
	default:
		return 0
	}
}

func (a DamageAssessment) Validate(caseID string) error {
	v := &ValidationError{}
	if a.CaseID != "" && a.CaseID != caseID {
		v.Add("case_id", "与当前事项不一致")
	}
	if a.Severity != SeverityMinor && a.Severity != SeverityModerate && a.Severity != SeveritySevere {
		v.Add("severity", "必须为 minor、moderate 或 severe")
	}
	if len(cleanList(a.Locations)) == 0 {
		v.Add("locations", "至少记录一个损伤部位")
	}
	if len(cleanList(a.Symptoms)) == 0 {
		v.Add("symptoms", "至少记录一个损伤表现")
	}
	if len(cleanList(a.ProbableCauses)) == 0 {
		v.Add("probable_causes", "至少记录一个劣化原因")
	}
	if len(cleanList(a.TreatmentGoals)) == 0 {
		v.Add("treatment_goals", "至少记录一个保护目标")
	}
	if len(cleanList(a.NonInterventionLimits)) == 0 {
		v.Add("non_intervention_limits", "必须说明不可干预边界")
	}
	required(v, "assessor", a.Assessor)
	validateEvidence("evidence_refs", a.EvidenceRefs, true, v)
	seen := map[string]bool{}
	for i, part := range a.PartitionAssessments {
		field := "partition_assessments[" + itoa(i) + "]"
		part.Location = strings.TrimSpace(part.Location)
		if part.Location == "" {
			v.Add(field+".location", "损伤部位不能为空")
		}
		if seen[part.Location] {
			v.Add(field+".location", "损伤部位不能重复")
		}
		seen[part.Location] = true
		if severityRank(part.Severity) == 0 {
			v.Add(field+".severity", "局部损伤等级无效")
		}
		if strings.TrimSpace(part.ImpactScope) == "" {
			v.Add(field+".impact_scope", "必须说明影响范围")
		}
		if len(cleanList(part.Symptoms)) == 0 {
			v.Add(field+".symptoms", "至少记录一个局部损伤表现")
		}
		if len(cleanList(part.TreatmentGoals)) == 0 {
			v.Add(field+".treatment_goals", "必须关联保护目标")
		}
		if len(cleanList(part.Boundaries)) == 0 {
			v.Add(field+".boundaries", "必须说明不可干预边界")
		}
		for _, values := range []struct {
			name string
			list []string
		}{{"treatment_goals", part.TreatmentGoals}, {"boundaries", part.Boundaries}} {
			semantic := map[string]bool{}
			for _, value := range values.list {
				key := strings.ToLower(strings.TrimSpace(value))
				if key != "" && semantic[key] {
					v.Add(field+"."+values.name, "同一部位不能包含语义相同的重复项")
				}
				semantic[key] = true
			}
		}
	}
	return v.OrNil()
}

func (c *ConservationCase) ConfirmAssessment(expected int64, assessment DamageAssessment, now time.Time) error {
	if err := CheckRevision(c.Revision, expected); err != nil {
		return err
	}
	if err := requireStatus(c.Status, StatusPendingAssessment); err != nil {
		return err
	}
	assessment.CaseID = c.ID
	assessment.Locations = cleanList(assessment.Locations)
	assessment.Symptoms = cleanList(assessment.Symptoms)
	assessment.ProbableCauses = cleanList(assessment.ProbableCauses)
	assessment.TreatmentGoals = cleanList(assessment.TreatmentGoals)
	assessment.NonInterventionLimits = cleanList(assessment.NonInterventionLimits)
	if len(assessment.PartitionAssessments) == 0 && len(assessment.LocationAssessments) > 0 {
		assessment.PartitionAssessments = assessment.LocationAssessments
	}
	for i := range assessment.PartitionAssessments {
		if len(assessment.PartitionAssessments[i].TreatmentGoals) == 0 {
			assessment.PartitionAssessments[i].TreatmentGoals = assessment.PartitionAssessments[i].Goals
		}
		if len(assessment.PartitionAssessments[i].Boundaries) == 0 {
			assessment.PartitionAssessments[i].Boundaries = assessment.PartitionAssessments[i].NonInterventionLimits
		}
		assessment.PartitionAssessments[i].Location = strings.TrimSpace(assessment.PartitionAssessments[i].Location)
		assessment.PartitionAssessments[i].Symptoms = cleanList(assessment.PartitionAssessments[i].Symptoms)
		assessment.PartitionAssessments[i].TreatmentGoals = cleanList(assessment.PartitionAssessments[i].TreatmentGoals)
		assessment.PartitionAssessments[i].Boundaries = cleanList(assessment.PartitionAssessments[i].Boundaries)
	}
	assessment.Assessor = strings.TrimSpace(assessment.Assessor)
	for i := range assessment.EvidenceRefs {
		assessment.EvidenceRefs[i].ID = strings.TrimSpace(assessment.EvidenceRefs[i].ID)
		assessment.EvidenceRefs[i].Filename = strings.TrimSpace(assessment.EvidenceRefs[i].Filename)
		assessment.EvidenceRefs[i].MediaType = strings.TrimSpace(assessment.EvidenceRefs[i].MediaType)
		assessment.EvidenceRefs[i].Note = strings.TrimSpace(assessment.EvidenceRefs[i].Note)
	}
	baseErr := assessment.Validate(c.ID)
	if baseErr != nil || len(assessment.PartitionAssessments) > 0 {
		combined := &ValidationError{}
		if baseErr != nil {
			copyIssues(combined, baseErr)
		}
		if len(assessment.PartitionAssessments) > 0 {
			seen := map[string]bool{}
			known := map[string]bool{}
			for _, location := range c.DamageLocations {
				known[strings.TrimSpace(location)] = true
			}
			for _, part := range assessment.PartitionAssessments {
				seen[part.Location] = true
				if !known[part.Location] {
					combined.Add("partition_assessments", "包含未知损伤部位："+part.Location)
				}
			}
			for _, location := range c.DamageLocations {
				if !seen[strings.TrimSpace(location)] {
					combined.Add("partition_assessments", "未覆盖建档损伤部位："+strings.TrimSpace(location))
				}
			}
			max := SeverityMinor
			for _, part := range assessment.PartitionAssessments {
				if severityRank(part.Severity) > severityRank(max) {
					max = part.Severity
				}
			}
			if severityRank(assessment.Severity) < severityRank(max) {
				combined.Add("severity", "总体损伤等级低于局部最高等级")
			}
			assessment.HighestPartitionSeverity = max
		}
		if err := combined.OrNil(); err != nil {
			return err
		}
	}
	assessment.ConfirmedAt = now.UTC()
	c.Assessment = &assessment
	c.advance(StatusProposalDrafting, now)
	return nil
}
