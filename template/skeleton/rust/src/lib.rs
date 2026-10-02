//! Starting point created by sdd-kit (adopt --skeleton).

/// Returns the greeting for `name`.
pub fn greeting(name: &str) -> String {
    let name = if name.is_empty() { "mundo" } else { name };
    format!("Olá, {name}!")
}

#[cfg(test)]
mod tests {
    use super::greeting;

    #[test]
    fn greets_by_name_or_the_world() {
        assert_eq!(greeting(""), "Olá, mundo!");
        assert_eq!(greeting("Ana"), "Olá, Ana!");
    }
}
