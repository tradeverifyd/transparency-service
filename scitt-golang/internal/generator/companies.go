package generator

// Company represents a semiconductor supply chain participant
type Company struct {
	Name      string
	Role      string
	URN       string
	Directory string
}

// GenerateCompanies returns the 3 predefined company identities
func GenerateCompanies() []Company {
	return []Company{
		{
			Name:      "Pacific Silicon Foundry",
			Role:      "foundry",
			URN:       "urn:supply-chain:pacific-silicon-foundry",
			Directory: "pacific-silicon-foundry",
		},
		{
			Name:      "Apex Semiconductor Corp",
			Role:      "IDM",
			URN:       "urn:supply-chain:apex-semiconductor-corp",
			Directory: "apex-semiconductor-corp",
		},
		{
			Name:      "Quantum Chip Design",
			Role:      "fabless",
			URN:       "urn:supply-chain:quantum-chip-design",
			Directory: "quantum-chip-design",
		},
	}
}
