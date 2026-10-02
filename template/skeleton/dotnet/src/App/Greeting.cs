namespace App;

/// <summary>Starting point created by sdd-kit (adopt --skeleton).</summary>
public static class Greeting
{
    /// <summary>Returns the greeting for <paramref name="name"/>.</summary>
    public static string For(string name) => $"Olá, {(name.Length == 0 ? "mundo" : name)}!";
}
