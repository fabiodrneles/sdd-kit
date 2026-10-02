package app;

/** Ponto de partida criado pelo sdd-kit (adopt --skeleton). */
public final class Greeting {
  private Greeting() {}

  public static String of(String name) {
    return "Olá, " + (name == null || name.isEmpty() ? "mundo" : name) + "!";
  }
}
