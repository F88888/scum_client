package util

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// 获取当前用户名
func getCurrentUsername() (string, error) {
	if runtime.GOOS == "windows" {
		return os.Getenv("USERNAME"), nil
	}
	return os.Getenv("USER"), nil
}

// 获取SCUM配置文件路径
func getSCUMConfigPath() (string, error) {
	username, err := getCurrentUsername()
	if err != nil {
		return "", fmt.Errorf("获取用户名失败: %v", err)
	}

	if runtime.GOOS == "windows" {
		configPath := filepath.Join("C:", "Users", username, "AppData", "Local", "SCUM", "Saved", "Config", "WindowsNoEditor", "GameUserSettings.ini")
		return configPath, nil
	}

	return "", fmt.Errorf("仅支持Windows系统")
}

// SCUM配置文件内容
const scumConfig = `[/Script/Engine.GameUserSettings]
FullscreenMode=2
LastConfirmedFullscreenMode=2
bUseDynamicResolution=False
ResolutionSizeX=841
ResolutionSizeY=554
LastUserConfirmedResolutionSizeX=841
LastUserConfirmedResolutionSizeY=554
AudioQualityLevel=0
LastConfirmedAudioQualityLevel=0
DesiredScreenWidth=1280
bUseDesiredScreenHeight=False
DesiredScreenHeight=720
LastUserConfirmedDesiredScreenWidth=1280
LastUserConfirmedDesiredScreenHeight=720
LastRecommendedScreenWidth=-1.000000
LastRecommendedScreenHeight=-1.000000
LastCPUBenchmarkResult=-1.000000
LastGPUBenchmarkResult=-1.000000
LastGPUBenchmarkMultiplier=1.000000
bUseHDRDisplayOutput=False
HDRDisplayOutputNits=1000

[Game]
scum.IsTelemetrySet=1
scum.TelemetryLevel=3
scum.LastEntitlementFlags=
scum.IsFirstPlaySession=0
scum.Language=3
scum.NudityCensoring=1
scum.PINCensoring=0
scum.ShowSimpleTooltipOnHover=1
scum.ShowAdditionalItemInfoWithoutHover=1
scum.EnableDeena=1
scum.AutoStartFirstDeenaTask=1
scum.SurvivalTipLevel=1
scum.ShowAnnouncementMessages=1
scum.ShowMusicPlayerDisplay=0
scum.EnableAirplaneFlightAssist=0
scum.NametagMode=0
scum.ShowUnofficialServerWarning=1
scum.ShowCutItemWarning=1
scum.ShowAbandonQuestWarning=1
scum.ShowChatTimestamps=0
scum.AimDownSightsMode=0
scum.LastUserProfile=-1
scum.QuickAccessVisibilityPreference=0
scum.QuickAccessTransparency=0.500000
scum.LifeIndicatorVisibilityPreference=0
scum.LifeIndicatorTransparency=0.500000
scum.AutomaticParachuteOpening=1

[Mouse]
scum.InvertMouseY=0
scum.InvertAirplaneMouseY=0
scum.MouseSensitivityFP=50
scum.MouseSensitivityTP=50
scum.MouseSensitivityDTS=50
scum.MouseSensitivityScope=50
scum.MouseSensitivityLockpicking=50
scum.MouseSensitivityBombDefusal=50
scum.MouseSensitivityATM=50
scum.MouseSensitivityDrone=50
scum.MouseSensitivityPhone=50

[Video]
scum.Gamma=2.400000
scum.FirstPersonFOV=70.000000
scum.ThirdPersonFOV=70.000000
scum.FirstPersonDrivingFOV=70.000000
scum.ThirdPersonDrivingFOV=70.000000
scum.CameraBobbingIntensity=0

[Graphics]
scum.RenderScale=0.100000
scum.DLSSSuperResolution=0
scum.DLSSFrameGeneration=0
scum.Reflex=1
scum.FSR=0
scum.ShadowQuality=0
scum.PostProcessingQuality=0
scum.EffectsQuality=0
scum.TextureQuality=0
scum.TextureMemory=0
scum.ViewDistance=0
scum.SkeletalMeshLODBias=0
scum.FoliageQuality=0
scum.FogQuality=0
scum.MotionBlur=1
scum.ShadowPrecision=0
scum.ShadowResolution=0
scum.DistanceFieldShadows=0
scum.DistanceFieldAmbientOcclusion=0
scum.RefractionQuality=0
scum.TranslucencyVolumeBlur=0
scum.DepthOfFieldQuality=0
scum.LensFlareQuality=0
scum.ChromaticAbberation=0
scum.BloomQuality=0
scum.TonemapperQuality=0
scum.FilmGrain=0
scum.LightShafts=0
scum.SeparateTranslucencyPass=0
scum.CloudsQuality=0
scum.CloudShadowQuality=0
scum.FoliageLODDithering=0

[Sound]
scum.MasterVolume=100
scum.MusicVolume=50
scum.EffectsVolume=100
scum.UIVolume=100
scum.VoiceChatVolume=100
scum.VoicelineVolume=100
scum.SpeakerConfiguration=0
scum.RadioMode=1
scum.PushToTalk=1
scum.Enable3DAudio=0
scum.CardiophobiaMode=0

[ScalabilityGroups]
sg.ResolutionQuality=100.000000
sg.ViewDistanceQuality=3
sg.AntiAliasingQuality=3
sg.ShadowQuality=3
sg.PostProcessQuality=3
sg.TextureQuality=3
sg.EffectsQuality=3
sg.FoliageQuality=3
sg.ShadingQuality=3`

// ReplaceSCUMConfig 替换SCUM配置文件
func ReplaceSCUMConfig() error {
	configPath, err := getSCUMConfigPath()
	if err != nil {
		return fmt.Errorf("获取配置文件路径失败: %v", err)
	}

	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Printf("SCUM配置文件不存在: %s\n", configPath)
		return nil
	}

	// 备份原配置文件
	backupPath := configPath + ".backup"
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		originalData, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("读取原配置文件失败: %v", err)
		}

		if err := os.WriteFile(backupPath, originalData, 0644); err != nil {
			fmt.Printf("备份配置文件失败: %v\n", err)
		} else {
			fmt.Printf("已备份原配置文件到: %s\n", backupPath)
		}
	}

	// 写入新配置
	if err := os.WriteFile(configPath, []byte(scumConfig), 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}

	fmt.Printf("成功替换SCUM配置文件: %s\n", configPath)
	return nil
}
