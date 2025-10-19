package generator

import "time"

// Document interface for all supply chain document types
type Document interface {
	GetSubjectURN() string
	GetContentType() string
}

// BaseDocument contains fields common to all documents
type BaseDocument struct {
	DocumentID      string `json:"document_id"`
	DocumentType    string `json:"document_type"`
	URN             string `json:"urn"`
	Issuer          string `json:"issuer"`
	Timestamp       string `json:"timestamp"`
	ContentType     string `json:"content_type"`
	ContentLocation string `json:"content_location"`
}

// WaferBatch represents a wafer manufacturing batch document
type WaferBatch struct {
	BaseDocument
	LotNumber           string  `json:"lot_number"`
	WaferDiameterMM     int     `json:"wafer_diameter_mm"`
	ThicknessUM         int     `json:"thickness_um"`
	Material            string  `json:"material"`
	CrystalOrientation  string  `json:"crystal_orientation"`
	ResistivityOhmCM    float64 `json:"resistivity_ohm_cm"`
	ProducedDate        string  `json:"produced_date"`
}

func (w *WaferBatch) GetSubjectURN() string   { return w.URN }
func (w *WaferBatch) GetContentType() string  { return w.ContentType }

// MineralSourcing represents conflict mineral sourcing documentation
type MineralSourcing struct {
	BaseDocument
	MineralType    string  `json:"mineral_type"`
	OriginCountry  string  `json:"origin_country"`
	MineName       string  `json:"mine_name"`
	Certification  string  `json:"certification"`
	QuantityKG     float64 `json:"quantity_kg"`
	ConflictFree   bool    `json:"conflict_free"`
}

func (m *MineralSourcing) GetSubjectURN() string  { return m.URN }
func (m *MineralSourcing) GetContentType() string { return m.ContentType }

// ChipSpecification represents a semiconductor chip specification
type ChipSpecification struct {
	BaseDocument
	PartNumber         string  `json:"part_number"`
	ChipType           string  `json:"chip_type"`
	Cores              int     `json:"cores,omitempty"`
	Threads            int     `json:"threads,omitempty"`
	BaseFrequencyGHz   float64 `json:"base_frequency_ghz,omitempty"`
	BoostFrequencyGHz  float64 `json:"boost_frequency_ghz,omitempty"`
	TDPW              int     `json:"tdp_watts,omitempty"`
	ProcessNodeNM      int     `json:"process_node_nm"`
	NPUIncluded        bool    `json:"npu_included,omitempty"`
	NPUTOPS            float64 `json:"npu_tops,omitempty"`
}

func (c *ChipSpecification) GetSubjectURN() string  { return c.URN }
func (c *ChipSpecification) GetContentType() string { return c.ContentType }

// FirmwareManifest represents firmware version and checksums
type FirmwareManifest struct {
	BaseDocument
	FirmwareType      string `json:"firmware_type"`
	Version           string `json:"version"`
	SHA256            string `json:"sha256"`
	SigningAuthority  string `json:"signing_authority"`
	ReleaseDate       string `json:"release_date"`
	FileSizeBytes     int64  `json:"file_size_bytes"`
}

func (f *FirmwareManifest) GetSubjectURN() string  { return f.URN }
func (f *FirmwareManifest) GetContentType() string { return f.ContentType }

// SBOM represents a Software Bill of Materials in SPDX 2.3 format
type SBOM struct {
	SPDXVersion       string         `json:"spdxVersion"`
	DataLicense       string         `json:"dataLicense"`
	SPDXID            string         `json:"SPDXID"`
	DocumentName      string         `json:"documentName"`
	DocumentNamespace string         `json:"documentNamespace"`
	CreationInfo      CreationInfo   `json:"creationInfo"`
	Packages          []Package      `json:"packages"`
}

type CreationInfo struct {
	LicenseListVersion string   `json:"licenseListVersion"`
	Creators           []string `json:"creators"`
	Created            string   `json:"created"`
}

type Package struct {
	SPDXID         string `json:"SPDXID"`
	Name           string `json:"name"`
	VersionInfo    string `json:"versionInfo"`
	Supplier       string `json:"supplier"`
	FilesAnalyzed  bool   `json:"filesAnalyzed"`
}

func (s *SBOM) GetSubjectURN() string   { return s.DocumentNamespace }
func (s *SBOM) GetContentType() string  { return "application/spdx+json" }

// MemorySpecification represents memory module specifications
type MemorySpecification struct {
	BaseDocument
	PartNumber   string  `json:"part_number"`
	MemoryType   string  `json:"memory_type"`
	CapacityGB   int     `json:"capacity_gb"`
	SpeedMHz     int     `json:"speed_mhz"`
	CASLatency   int     `json:"cas_latency"`
	Voltage      float64 `json:"voltage"`
}

func (m *MemorySpecification) GetSubjectURN() string  { return m.URN }
func (m *MemorySpecification) GetContentType() string { return m.ContentType }

// AITrainingDataset represents AI training data provenance
type AITrainingDataset struct {
	BaseDocument
	DatasetName     string `json:"dataset_name"`
	Source          string `json:"source"`
	License         string `json:"license"`
	SizeGB          int    `json:"size_gb"`
	DataProvenance  string `json:"data_provenance"`
	Usage           string `json:"usage"`
	RecordCount     int64  `json:"record_count"`
	Modality        string `json:"modality"`
}

func (a *AITrainingDataset) GetSubjectURN() string  { return a.URN }
func (a *AITrainingDataset) GetContentType() string { return a.ContentType }

// AIModelSpecification represents AI model architecture and parameters
type AIModelSpecification struct {
	BaseDocument
	ModelName          string  `json:"model_name"`
	Architecture       string  `json:"architecture"`
	ParametersMillions int     `json:"parameters_millions"`
	Quantization       string  `json:"quantization"`
	InferenceLatencyMS int     `json:"inference_latency_ms"`
	TargetHardware     string  `json:"target_hardware"`
	InferenceTOPS      float64 `json:"inference_tops"`
}

func (a *AIModelSpecification) GetSubjectURN() string  { return a.URN }
func (a *AIModelSpecification) GetContentType() string { return a.ContentType }

// CVEDocument represents a vulnerability disclosure
type CVEDocument struct {
	BaseDocument
	VulnerabilityID   string   `json:"vulnerability_id"` // CVE-XXXX-XXXX or CWE-XXX
	CWEID             string   `json:"cwe_id,omitempty"`
	Title             string   `json:"title"`
	CVSSScore         float64  `json:"cvss_score"`
	Severity          string   `json:"severity"`
	AffectedComponent string   `json:"affected_component"`
	AffectedVersions  []string `json:"affected_versions"`
	PatchedVersion    string   `json:"patched_version,omitempty"`
	DisclosureDate    string   `json:"disclosure_date"`
	Description       string   `json:"description"`
}

func (c *CVEDocument) GetSubjectURN() string  { return c.URN }
func (c *CVEDocument) GetContentType() string { return c.ContentType }

// LogisticsTracking represents shipment tracking information
type LogisticsTracking struct {
	BaseDocument
	ShipmentID    string `json:"shipment_id"`
	Origin        string `json:"origin"`
	Destination   string `json:"destination"`
	DepartureDate string `json:"departure_date"`
	ArrivalDate   string `json:"arrival_date,omitempty"`
	Contents      string `json:"contents"`
	Status        string `json:"status"`
}

func (l *LogisticsTracking) GetSubjectURN() string  { return l.URN }
func (l *LogisticsTracking) GetContentType() string { return l.ContentType }

// Helper function to get current timestamp in ISO 8601 format
func GetTimestamp() string {
	return time.Now().UTC().Format(time.RFC3339)
}
