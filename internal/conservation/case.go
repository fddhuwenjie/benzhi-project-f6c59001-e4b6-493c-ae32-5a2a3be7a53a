package conservation

import (
	"reflect"
	"strings"
	"time"
)

type ConservationCase struct {
	ID                     string                  `json:"id"`
	Revision               int64                   `json:"revision"`
	ShelfMark              string                  `json:"shelf_mark"`
	Title                  string                  `json:"title"`
	VersionIdentifier      string                  `json:"version_identifier"`
	SupportMaterial        string                  `json:"support_material"`
	CarrierCharacteristics string                  `json:"carrier_characteristics"`
	DamageLocations        []string                `json:"damage_locations"`
	InitialEvidence        []EvidenceRef           `json:"initial_evidence"`
	ResponsibleConservator string                  `json:"responsible_conservator"`
	Status                 Status                  `json:"status"`
	Assessment             *DamageAssessment       `json:"assessment,omitempty"`
	Proposal               *TreatmentProposal      `json:"proposal,omitempty"`
	Reviews                []PeerReview            `json:"reviews,omitempty"`
	Trial                  *TrialRecord            `json:"trial,omitempty"`
	ProposalHistory        []ProposalVersionRecord `json:"proposal_history,omitempty"`
	ReviewRoster           *ReviewRoster           `json:"review_roster,omitempty"`
	TrialBatches           []TrialRecord           `json:"trial_batches,omitempty"`
	ArchivePrecheck        *PrecheckResult         `json:"archive_precheck,omitempty"`
	CreatedAt              time.Time               `json:"created_at"`
	UpdatedAt              time.Time               `json:"updated_at"`
	ArchivedAt             *time.Time              `json:"archived_at,omitempty"`
}

type NewCase struct {
	ID                     string
	ShelfMark              string
	Title                  string
	VersionIdentifier      string
	SupportMaterial        string
	CarrierCharacteristics string
	DamageLocations        []string
	InitialEvidence        []EvidenceRef
	ResponsibleConservator string
}

func CreateCase(in NewCase, now time.Time) (*ConservationCase, error) {
	v := &ValidationError{}
	required(v, "id", in.ID)
	required(v, "shelf_mark", in.ShelfMark)
	required(v, "title", in.Title)
	required(v, "responsible_conservator", in.ResponsibleConservator)
	if err := v.OrNil(); err != nil {
		return nil, err
	}
	evidence := append([]EvidenceRef(nil), in.InitialEvidence...)
	for i := range evidence {
		evidence[i].ID = strings.TrimSpace(evidence[i].ID)
		evidence[i].Filename = strings.TrimSpace(evidence[i].Filename)
		evidence[i].MediaType = strings.TrimSpace(evidence[i].MediaType)
		evidence[i].Note = strings.TrimSpace(evidence[i].Note)
	}
	return &ConservationCase{
		ID: strings.TrimSpace(in.ID), Revision: 1, ShelfMark: strings.TrimSpace(in.ShelfMark),
		Title: strings.TrimSpace(in.Title), VersionIdentifier: strings.TrimSpace(in.VersionIdentifier),
		SupportMaterial: strings.TrimSpace(in.SupportMaterial), CarrierCharacteristics: strings.TrimSpace(in.CarrierCharacteristics),
		DamageLocations: cleanList(in.DamageLocations), InitialEvidence: evidence,
		ResponsibleConservator: strings.TrimSpace(in.ResponsibleConservator), Status: StatusDraft,
		CreatedAt: now.UTC(), UpdatedAt: now.UTC(),
	}, nil
}

func (c *ConservationCase) ValidateForSubmission() error {
	v := &ValidationError{}
	required(v, "version_identifier", c.VersionIdentifier)
	required(v, "support_material", c.SupportMaterial)
	required(v, "carrier_characteristics", c.CarrierCharacteristics)
	if len(c.DamageLocations) == 0 {
		v.Add("damage_locations", "至少记录一个损伤部位")
	}
	validateEvidence("initial_evidence", c.InitialEvidence, true, v)
	return v.OrNil()
}

func (c *ConservationCase) SubmitForAssessment(expected int64, now time.Time) error {
	if err := CheckRevision(c.Revision, expected); err != nil {
		return err
	}
	if err := requireStatus(c.Status, StatusDraft); err != nil {
		return err
	}
	if err := c.ValidateForSubmission(); err != nil {
		return err
	}
	c.advance(StatusPendingAssessment, now)
	return nil
}

type DraftRevision struct {
	Title                  string        `json:"title"`
	VersionIdentifier      string        `json:"version_identifier"`
	CarrierCharacteristics string        `json:"carrier_characteristics"`
	DamageLocations        []string      `json:"damage_locations"`
	InitialEvidence        []EvidenceRef `json:"initial_evidence"`
}

func (c *ConservationCase) ReviseDraft(expected int64, revision DraftRevision, now time.Time) (map[string]any, error) {
	if err := CheckRevision(c.Revision, expected); err != nil {
		return nil, err
	}
	if err := requireStatus(c.Status, StatusDraft); err != nil {
		return nil, err
	}
	if strings.TrimSpace(revision.Title) == "" {
		revision.Title = c.Title
	}
	revision.Title = strings.TrimSpace(revision.Title)
	revision.VersionIdentifier = strings.TrimSpace(revision.VersionIdentifier)
	revision.CarrierCharacteristics = strings.TrimSpace(revision.CarrierCharacteristics)
	revision.DamageLocations = cleanList(revision.DamageLocations)
	for i := range revision.InitialEvidence {
		revision.InitialEvidence[i].ID = strings.TrimSpace(revision.InitialEvidence[i].ID)
		revision.InitialEvidence[i].Filename = strings.TrimSpace(revision.InitialEvidence[i].Filename)
		revision.InitialEvidence[i].MediaType = strings.TrimSpace(revision.InitialEvidence[i].MediaType)
	}
	v := &ValidationError{}
	required(v, "title", revision.Title)
	required(v, "version_identifier", revision.VersionIdentifier)
	required(v, "carrier_characteristics", revision.CarrierCharacteristics)
	if len(revision.DamageLocations) == 0 {
		v.Add("damage_locations", "至少记录一个损伤部位")
	}
	validateEvidence("initial_evidence", revision.InitialEvidence, true, v)
	if err := v.OrNil(); err != nil {
		return nil, err
	}
	changes := map[string]any{}
	if c.Title != revision.Title {
		changes["title"] = map[string]string{"from": c.Title, "to": revision.Title}
		c.Title = revision.Title
	}
	if c.VersionIdentifier != revision.VersionIdentifier {
		changes["version_identifier"] = map[string]string{"from": c.VersionIdentifier, "to": revision.VersionIdentifier}
		c.VersionIdentifier = revision.VersionIdentifier
	}
	if c.CarrierCharacteristics != revision.CarrierCharacteristics {
		changes["carrier_characteristics"] = map[string]string{"from": c.CarrierCharacteristics, "to": revision.CarrierCharacteristics}
		c.CarrierCharacteristics = revision.CarrierCharacteristics
	}
	if strings.Join(c.DamageLocations, "\x00") != strings.Join(revision.DamageLocations, "\x00") {
		changes["damage_locations"] = map[string]any{"from": c.DamageLocations, "to": revision.DamageLocations}
		c.DamageLocations = revision.DamageLocations
	}
	if !reflect.DeepEqual(c.InitialEvidence, revision.InitialEvidence) {
		changes["initial_evidence"] = map[string]any{"from": c.InitialEvidence, "to": revision.InitialEvidence}
		c.InitialEvidence = revision.InitialEvidence
	}
	c.Revision++
	c.UpdatedAt = now.UTC()
	return changes, nil
}

func (c *ConservationCase) advance(status Status, now time.Time) {
	c.Status = status
	c.Revision++
	c.UpdatedAt = now.UTC()
}

func required(v *ValidationError, field, value string) {
	if strings.TrimSpace(value) == "" {
		v.Add(field, "不能为空")
	}
}

func cleanList(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
		}
	}
	return result
}
