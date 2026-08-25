package workflow

import "benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/conservation"

type CommandMeta struct {
	RequestID        string `json:"request_id"`
	ExpectedRevision int64  `json:"expected_revision"`
	Actor            Actor  `json:"actor"`
}

type CreateCaseCommand struct {
	RequestID              string                     `json:"request_id"`
	Actor                  Actor                      `json:"actor"`
	ID                     string                     `json:"id,omitempty"`
	ShelfMark              string                     `json:"shelf_mark"`
	Title                  string                     `json:"title"`
	VersionIdentifier      string                     `json:"version_identifier"`
	SupportMaterial        string                     `json:"support_material"`
	CarrierCharacteristics string                     `json:"carrier_characteristics"`
	DamageLocations        []string                   `json:"damage_locations"`
	InitialEvidence        []conservation.EvidenceRef `json:"initial_evidence"`
	ResponsibleConservator string                     `json:"responsible_conservator"`
}

type AssessmentCommand struct {
	CommandMeta
	Assessment conservation.DamageAssessment `json:"assessment"`
}

type DraftRevisionCommand struct {
	CommandMeta
	Revision               conservation.DraftRevision `json:"revision"`
	Draft                  conservation.DraftRevision `json:"draft,omitempty"`
	Title                  string                     `json:"title,omitempty"`
	VersionIdentifier      string                     `json:"version_identifier,omitempty"`
	CarrierCharacteristics string                     `json:"carrier_characteristics,omitempty"`
	DamageLocations        []string                   `json:"damage_locations,omitempty"`
	InitialEvidence        []conservation.EvidenceRef `json:"initial_evidence,omitempty"`
}

type RosterCommand struct {
	CommandMeta
	Roster conservation.ReviewRoster `json:"roster"`
}

type ProposalCommand struct {
	CommandMeta
	Proposal      conservation.TreatmentProposal `json:"proposal"`
	ResponseNotes map[string]string              `json:"response_notes,omitempty"`
}

type ReviewCommand struct {
	CommandMeta
	Review conservation.PeerReview `json:"review"`
}

type TrialCommand struct {
	CommandMeta
	Trial conservation.TrialRecord `json:"trial"`
}

type ArchiveCommand struct {
	CommandMeta
	ConfirmedHints []string `json:"confirmed_hints,omitempty"`
}
