from app import greeting


def test_greeting() -> None:
    assert greeting() == "Olá, mundo!"
    assert greeting("Ana") == "Olá, Ana!"
