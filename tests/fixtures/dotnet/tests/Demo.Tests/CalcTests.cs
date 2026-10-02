namespace Demo.Tests;

public class CalcTests
{
    [Fact]
    public void Sums() => Assert.Equal(5, Calc.Sum(2, 3));
}
