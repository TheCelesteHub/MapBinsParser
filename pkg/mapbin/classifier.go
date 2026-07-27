package mapbin

import "regexp"

var (
	SuffixDenyPattern = regexp.MustCompile(`(?i)(trigger|controller|gate|door|block|spawner|respawnpoint|toflag|seed|jar|cabin|switch|field|activator)$`)
	FakeDenyPattern   = regexp.MustCompile(`(?i)(troll|fake)`)
	HeartPattern      = regexp.MustCompile(`(?i)(crystalheart|heartgem)`)
	GoldenPattern     = regexp.MustCompile(`(?i)golden.?berry`)
	BerryPattern      = regexp.MustCompile(`(?i)(berry|berries|strawberr)`)
	PlatinumPattern   = regexp.MustCompile(`(?i)platinum`)
	SpawnPattern      = regexp.MustCompile(`(?i)(player|spawn)`)
	HazardPattern     = regexp.MustCompile(`(?i)(spike|spring|spinner|lightning)`)
)

var ExactDeny = map[string]bool{
	"vitellary/keyberry":                       true,
	"FactoryHelper/MachineHeart":               true,
	"CommunalHelper/CustomSummitGem":           true,
	"ScugHelper/SquareGem":                     true,
	"HeliosHelper/SolHeartsideSwitch":          true,
	"IntoTheJungleCodeMod/LanternHeartSpawner": true,
}

var ExactHearts = map[string]bool{
	"blackGem":            true,
	"birdForsakenCityGem": true,
	"dreamHeartGem":       true,
}

func CountEntity(counts *MapCollectibleCounts, name string, moon bool, winged bool) {
	if FakeDenyPattern.MatchString(name) || SuffixDenyPattern.MatchString(name) || ExactDeny[name] {
		return
	}

	switch {
	case name == "CollabUtils2/MiniHeart":
		counts.MiniHearts++
	case ExactHearts[name] || HeartPattern.MatchString(name):
		counts.Hearts++
	case name == "CollabUtils2/SilverBerry":
		counts.Silver++
	case name == "CollabUtils2/SpeedBerry":
		counts.Speed++
	case name == "CollabUtils2/RainbowBerry":
		counts.Rainbow++
	case PlatinumPattern.MatchString(name) && BerryPattern.MatchString(name):
		counts.Platinum++
	case GoldenPattern.MatchString(name):
		if winged {
			counts.WingedGolden++
		} else {
			counts.Golden++
		}
	case BerryPattern.MatchString(name):
		if moon {
			counts.Moon++
		} else {
			counts.Red++
		}
	}
}

func IsCollectibleEntity(name string, moon bool, winged bool) bool {
	if FakeDenyPattern.MatchString(name) || SuffixDenyPattern.MatchString(name) || ExactDeny[name] {
		return false
	}
	switch {
	case name == "CollabUtils2/MiniHeart",
		ExactHearts[name], HeartPattern.MatchString(name),
		name == "CollabUtils2/SilverBerry",
		name == "CollabUtils2/SpeedBerry",
		name == "CollabUtils2/RainbowBerry",
		GoldenPattern.MatchString(name),
		BerryPattern.MatchString(name):
		return true
	case PlatinumPattern.MatchString(name) && BerryPattern.MatchString(name):
		return true
	}
	return false
}
