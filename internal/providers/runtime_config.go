package providers

type RuntimeConfig struct {
	MockDataFiles   []string
	AirAsia         ProviderRuntimeConfig
	BatikAir        ProviderRuntimeConfig
	GarudaIndonesia ProviderRuntimeConfig
	LionAir         ProviderRuntimeConfig
}

type ProviderRuntimeConfig struct {
	DelayMS     int
	FailureRate int
}
