package stats

import "testing"

func TestD20(t *testing.T) {
	// Roll many times and verify results are in range
	for i := 0; i < 100; i++ {
		result := D20()
		if result < 1 || result > 20 {
			t.Errorf("D20() = %d, expected 1-20", result)
		}
	}
}

func TestD12(t *testing.T) {
	for i := 0; i < 100; i++ {
		result := D12()
		if result < 1 || result > 12 {
			t.Errorf("D12() = %d, expected 1-12", result)
		}
	}
}

func TestD10(t *testing.T) {
	for i := 0; i < 100; i++ {
		result := D10()
		if result < 1 || result > 10 {
			t.Errorf("D10() = %d, expected 1-10", result)
		}
	}
}

func TestD8(t *testing.T) {
	for i := 0; i < 100; i++ {
		result := D8()
		if result < 1 || result > 8 {
			t.Errorf("D8() = %d, expected 1-8", result)
		}
	}
}

func TestD6(t *testing.T) {
	for i := 0; i < 100; i++ {
		result := D6()
		if result < 1 || result > 6 {
			t.Errorf("D6() = %d, expected 1-6", result)
		}
	}
}

func TestD4(t *testing.T) {
	for i := 0; i < 100; i++ {
		result := D4()
		if result < 1 || result > 4 {
			t.Errorf("D4() = %d, expected 1-4", result)
		}
	}
}

func TestRoll(t *testing.T) {
	// Test 1d6
	for i := 0; i < 100; i++ {
		result := Roll(1, 6)
		if result < 1 || result > 6 {
			t.Errorf("Roll(1, 6) = %d, expected 1-6", result)
		}
	}

	// Test 2d6 (range 2-12)
	for i := 0; i < 100; i++ {
		result := Roll(2, 6)
		if result < 2 || result > 12 {
			t.Errorf("Roll(2, 6) = %d, expected 2-12", result)
		}
	}

	// Test 3d6 (range 3-18)
	for i := 0; i < 100; i++ {
		result := Roll(3, 6)
		if result < 3 || result > 18 {
			t.Errorf("Roll(3, 6) = %d, expected 3-18", result)
		}
	}

	// Test edge case: 0 dice
	result := Roll(0, 6)
	if result != 0 {
		t.Errorf("Roll(0, 6) = %d, expected 0", result)
	}
}

func TestRollWithBonus(t *testing.T) {
	// Test 1d6+2 (range 3-8)
	for i := 0; i < 100; i++ {
		result := RollWithBonus(1, 6, 2)
		if result < 3 || result > 8 {
			t.Errorf("RollWithBonus(1, 6, 2) = %d, expected 3-8", result)
		}
	}

	// Test 2d6-1 (range 1-11)
	for i := 0; i < 100; i++ {
		result := RollWithBonus(2, 6, -1)
		if result < 1 || result > 11 {
			t.Errorf("RollWithBonus(2, 6, -1) = %d, expected 1-11", result)
		}
	}
}
