package deck

// Environmental plaque gettext msgids per functional layer (Story 5.1 — FR26).
// Placement rules: see specs/environmental-signage.md and setup.ApplyEnvironmentalSignage.
func EnvironmentalPlaqueKeys(t Type) []string {
	switch t {
	case Habitation:
		return []string{
			"ENV_PLAQUE_HAB_ATM",
			"ENV_PLAQUE_HAB_QUIET",
			"ENV_PLAQUE_HAB_PRESSURE",
			"ENV_PLAQUE_HAB_CREWLOG",
			"ENV_PLAQUE_HAB_LIFE_BUS",
		}
	case Research:
		return []string{
			"ENV_PLAQUE_RES_SAMPLE",
			"ENV_PLAQUE_RES_CONTAIN",
			"ENV_PLAQUE_RES_CALIB",
			"ENV_PLAQUE_RES_ARCHIVE",
		}
	case Logistics:
		return []string{
			"ENV_PLAQUE_LOG_ROUTE",
			"ENV_PLAQUE_LOG_MANIFEST",
			"ENV_PLAQUE_LOG_TRANSFER",
			"ENV_PLAQUE_LOG_WEIGHT",
		}
	case PowerDistribution:
		return []string{
			"ENV_PLAQUE_PWR_FEED",
			"ENV_PLAQUE_PWR_RELAY",
			"ENV_PLAQUE_PWR_PHASE",
			"ENV_PLAQUE_PWR_AUX",
		}
	case EmergencySystems:
		return []string{
			"ENV_PLAQUE_EMERG_BEACON",
			"ENV_PLAQUE_EMERG_SHELTER",
			"ENV_PLAQUE_EMERG_OVERRIDE",
			"ENV_PLAQUE_EMERG_DRILL",
		}
	case CoreInfrastructure:
		return []string{
			"ENV_PLAQUE_CORE_SPINE",
			"ENV_PLAQUE_CORE_MON",
			"ENV_PLAQUE_CORE_AUDIT",
			"ENV_PLAQUE_CORE_ARCHIVE",
		}
	default:
		return []string{"ENV_PLAQUE_GEN_ROUTE"}
	}
}
