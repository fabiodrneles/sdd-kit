package price

import (
	"errors"
	"testing"
)

func TestCategory(t *testing.T) {
	cases := map[int]string{0: "criança", 11: "criança", 12: "jovem", 17: "jovem", 18: "adulto", 59: "adulto", 60: "idoso"}
	for age, want := range cases {
		if got, err := Category(age); err != nil || got != want {
			t.Errorf("Category(%d) = %q, %v; quer %q", age, got, err, want)
		}
	}
	if _, err := Category(-1); !errors.Is(err, ErrAge) {
		t.Errorf("idade negativa: %v", err)
	}
}

func TestPrice(t *testing.T) {
	cases := []struct {
		age     int
		student bool
		want    int
	}{{5, false, 500}, {15, false, 800}, {30, false, 1000}, {70, false, 600}, {30, true, 900}, {5, true, 450}}
	for _, c := range cases {
		if got, err := Price(1000, c.age, c.student); err != nil || got != c.want {
			t.Errorf("Price(1000, %d, %v) = %d, %v; quer %d", c.age, c.student, got, err, c.want)
		}
	}
	if _, err := Price(1000, -1, false); !errors.Is(err, ErrAge) {
		t.Errorf("idade negativa: %v", err)
	}
}
