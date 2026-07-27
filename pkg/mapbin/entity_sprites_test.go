package mapbin

import "testing"

func TestBerryTexture_SelectsCorrectVariant(t *testing.T) {
	cases := []struct {
		name     string
		ent      *EntityData
		expected string
	}{
		{"plain", &EntityData{}, "collectables/strawberry/normal00"},
		{"winged", &EntityData{Winged: true}, "collectables/strawberry/wings01"},
		{"hasNodes", &EntityData{HasNodes: true}, "collectables/ghostberry/idle00"},
		{"winged+hasNodes", &EntityData{Winged: true, HasNodes: true}, "collectables/ghostberry/wings01"},
		{"moon", &EntityData{Moon: true}, "collectables/moonBerry/normal00"},
		{"moon+winged", &EntityData{Moon: true, Winged: true}, "collectables/moonBerry/ghost00"},
		{"moon+hasNodes", &EntityData{Moon: true, HasNodes: true}, "collectables/moonBerry/ghost00"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := berryTexture(c.ent); got != c.expected {
				t.Errorf("berryTexture(%+v) = %q, want %q", c.ent, got, c.expected)
			}
		})
	}
}

func TestSpikeDirection_ParsesNameSuffix(t *testing.T) {
	cases := []struct {
		name     string
		expected string
	}{
		{"spikesUp", "up"},
		{"spikesDown", "down"},
		{"spikesLeft", "left"},
		{"spikesRight", "right"},
		{"triggerSpikesUp", "up"},
		{"strawberry", ""},
	}
	for _, c := range cases {
		if got := spikeDirection(c.name); got != c.expected {
			t.Errorf("spikeDirection(%q) = %q, want %q", c.name, got, c.expected)
		}
	}
}
