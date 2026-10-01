package demo;

import static org.junit.jupiter.api.Assertions.assertEquals;

import org.junit.jupiter.api.Test;

class SumTest {
  @Test
  void sum() {
    assertEquals(5, Sum.of(2, 3));
  }
}
