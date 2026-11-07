import org.jetbrains.kotlin.gradle.dsl.JvmTarget
import org.jetbrains.kotlin.gradle.tasks.KotlinCompile

plugins {
    id("com.android.library")
}

android {
    compileSdk = pinnedVersionOf("android-platform").substringBefore('.').toInt()
    buildToolsVersion = pinnedVersionOf("android-build-tools")
    ndkVersion = pinnedVersionOf("android-ndk")
    testBuildType = "release"

    defaultConfig {
        minSdk = APP_MIN_SDK
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
