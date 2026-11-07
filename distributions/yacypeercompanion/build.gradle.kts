val ktlint = configurations.create("ktlint")

dependencies {
    ktlint("com.pinterest.ktlint:ktlint-cli:1.8.0") {
        attributes { attribute(Bundling.BUNDLING_ATTRIBUTE, objects.named(Bundling.EXTERNAL)) }
    }
}

val kotlinSources = listOf("*/src/**/*.kt", "*.kts", "*/*.kts", "build-logic/src/**/*.kts", "!**/build/**")

tasks.register<JavaExec>("ktlintCheck") {
    description = "Checks the Kotlin code style (.editorconfig) of every module."
    group = "verification"
    classpath = ktlint
    mainClass.set("com.pinterest.ktlint.Main")
    args = kotlinSources
}

tasks.register<JavaExec>("ktlintFormat") {
    description = "Fixes Kotlin code style findings where ktlint can."
    group = "formatting"
    classpath = ktlint
    mainClass.set("com.pinterest.ktlint.Main")
    jvmArgs("--add-opens=java.base/java.lang=ALL-UNNAMED")
    args = listOf("-F") + kotlinSources
}
