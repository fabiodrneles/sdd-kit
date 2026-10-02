namespace App.Tests;

public class GreetingTests
{
    [Theory]
    [InlineData("", "Olá, mundo!")]
    [InlineData("Ana", "Olá, Ana!")]
    public void GreetsByNameOrTheWorld(string name, string expected) =>
        Assert.Equal(expected, Greeting.For(name));
}
