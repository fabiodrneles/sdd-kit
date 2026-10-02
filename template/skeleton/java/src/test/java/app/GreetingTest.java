package app;

import static org.junit.jupiter.api.Assertions.assertEquals;

import org.junit.jupiter.api.Test;

class GreetingTest {
  @Test
  void greeting() {
    assertEquals("Olá, mundo!", Greeting.of(""));
    assertEquals("Olá, mundo!", Greeting.of(null));
    assertEquals("Olá, Ana!", Greeting.of("Ana"));
  }
}
