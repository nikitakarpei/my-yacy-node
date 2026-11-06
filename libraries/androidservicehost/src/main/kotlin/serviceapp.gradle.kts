import org.jetbrains.kotlin.gradle.dsl.JvmTarget
import org.jetbrains.kotlin.gradle.tasks.KotlinCompile

plugins {
    id("com.android.application")
}

val serviceName = name.removeSuffix("android")
val appMinSdk = 28
val goLibrary = layout.buildDirectory.file("go/$serviceName.aar")

android {
    namespace = "io.github.nikitakarpei.$serviceName"
    compileSdk = pinnedVersionOf("android-platform").substringBefore('.').toInt()
    buildToolsVersion = pinnedVersionOf("android-build-tools")
    ndkVersion = pinnedVersionOf("android-ndk")

    defaultConfig {
        applicationId = "io.github.nikitakarpei.$serviceName"
        minSdk = appMinSdk
        targetSdk = 36
    }

    sourceSets {
        getByName("main") {
            kotlin.directories.add(repositoryFile("libraries/androidservicehost/host/java").path)
            res.directories.add(repositoryFile("libraries/androidservicehost/host/res").path)
        }
    }

    buildTypes {
        release {
            isMinifyEnabled = false
            isDebuggable = false
            vcsInfo.include = false
        }
    }

    dependenciesInfo {
        includeInApk = false
        includeInBundle = false
    }

    androidResources {
        noCompress += ""
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    lint {
        warningsAsErrors = true
        abortOnError = true
        checkReleaseBuilds = true
        disable += listOf("GradleDependency", "NewerVersionAvailable", "AndroidGradlePluginVersion", "OldTargetApi")
    }
}

tasks.matching { it.name.contains("ArtProfile") }.configureEach { enabled = false }

tasks.withType<KotlinCompile>().configureEach {
    compilerOptions {
        jvmTarget.set(JvmTarget.JVM_17)
        allWarningsAsErrors.set(true)
    }
}

val bindGo =
    tasks.register<Exec>("bindGo") {
        description = "Binds the Go package $serviceName into an Android library with gomobile."
        group = "build"
        workingDir = projectDir
        commandLine(
            repositoryFile("libraries/androidservicehost/bind-go").path,
            serviceName,
            goLibrary.get().asFile.absolutePath,
            appMinSdk.toString(),
            pinnedVersionOf("android-ndk"),
        )
        outputs.file(goLibrary)
        outputs.upToDateWhen { false }
    }

val ktlint = configurations.create("ktlint")

dependencies {
    "implementation"(files(goLibrary).builtBy(bindGo))
    ktlint("com.pinterest.ktlint:ktlint-cli:1.8.0") {
        attributes { attribute(Bundling.BUNDLING_ATTRIBUTE, objects.named(Bundling.EXTERNAL)) }
    }
}

val kotlinSources =
    listOf("src/**/*.kt", "*.kts", "../../libraries/androidservicehost/host/**/*.kt") +
        listOf("../../libraries/androidservicehost/src/**/*.kt", "../../libraries/androidservicehost/src/**/*.kts") +
        listOf("../../libraries/androidservicehost/*.kts")

tasks.register<JavaExec>("ktlintCheck") {
    description = "Checks the Kotlin code style (.editorconfig) of the app and of its shared host."
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
