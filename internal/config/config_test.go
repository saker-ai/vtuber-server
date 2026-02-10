package config

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestTopLevelXiaoZhiFeatureAECExplicitFromConfig(t *testing.T) {
	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(strings.NewReader("xiaozhi_feature_aec: false\n")); err != nil {
		t.Fatalf("ReadConfig error: %v", err)
	}

	if !isTopLevelXiaoZhiFeatureAECExplicit(v) {
		t.Fatal("isTopLevelXiaoZhiFeatureAECExplicit=false, want true")
	}
	if isSystemXiaoZhiFeatureAECExplicit(v) {
		t.Fatal("isSystemXiaoZhiFeatureAECExplicit=true, want false")
	}
}

func TestSystemXiaoZhiFeatureAECExplicitFromConfig(t *testing.T) {
	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(strings.NewReader("system_config:\n  xiaozhi_feature_aec: false\n")); err != nil {
		t.Fatalf("ReadConfig error: %v", err)
	}

	if isTopLevelXiaoZhiFeatureAECExplicit(v) {
		t.Fatal("isTopLevelXiaoZhiFeatureAECExplicit=true, want false")
	}
	if !isSystemXiaoZhiFeatureAECExplicit(v) {
		t.Fatal("isSystemXiaoZhiFeatureAECExplicit=false, want true")
	}
}

func TestXiaoZhiFeatureAECExplicitFromEnv(t *testing.T) {
	v := viper.New()

	t.Setenv("MIO_XIAOZHI_FEATURE_AEC", "false")
	if !isTopLevelXiaoZhiFeatureAECExplicit(v) {
		t.Fatal("isTopLevelXiaoZhiFeatureAECExplicit=false with env, want true")
	}

	t.Setenv("MIO_SYSTEM_CONFIG_XIAOZHI_FEATURE_AEC", "false")
	if !isSystemXiaoZhiFeatureAECExplicit(v) {
		t.Fatal("isSystemXiaoZhiFeatureAECExplicit=false with env, want true")
	}
}
