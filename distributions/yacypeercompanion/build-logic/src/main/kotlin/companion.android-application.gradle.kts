import com.android.build.api.artifact.SingleArtifact
import org.jetbrains.kotlin.gradle.dsl.JvmTarget
import org.jetbrains.kotlin.gradle.tasks.KotlinCompile

plugins {
    id("com.android.application")
}

android {
    compileSdk = pinnedVersionOf("android-platform").substringBefore('.').toInt()
    buildToolsVersion = pinnedVersionOf("android-build-tools")
    ndkVersion = pinnedVersionOf("android-ndk")
    testBuildType = "release"

    defaultConfig {
        minSdk = APP_MIN_SDK
        targetSdk = 36
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

    packaging {
        resources.pickFirsts += listOf("META-INF/LICENSE.md", "META-INF/LICENSE.txt", "META-INF/LICENSE")
        resources.excludes += listOf("META-INF/versions/*/OSGI-INF/MANIFEST.MF", "META-INF/versions/*/module-info.class")
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

androidComponents {
    onVariants(selector().withBuildType("release")) { variant ->
        tasks.register<AttackSurfaceCheck>("check${variant.name.replaceFirstChar(Char::titlecase)}AttackSurface") {
            description = "Checks that the libraries widen no attack surface of the APK and that its platform protections stay on."
            group = "verification"
            appManifest.set(layout.projectDirectory.file("src/main/AndroidManifest.xml"))
            apkManifest.set(variant.artifacts.get(SingleArtifact.MERGED_MANIFEST))
            apkDirectory.set(variant.artifacts.get(SingleArtifact.APK))
            namespace.set(variant.namespace)
        }
        val libraryLockfile = layout.buildDirectory.file("${variant.name}-libraries/gradle.lockfile")
        val writeLibraryLockfile =
            tasks.register<LibraryLockfileWriter>("write${variant.name.replaceFirstChar(Char::titlecase)}LibraryLockfile") {
                description = "Writes the libraries of the APK as a Gradle lockfile."
                rootComponent.set(variant.runtimeConfiguration.incoming.resolutionResult.rootComponent)
                configurationName.set(variant.runtimeConfiguration.name)
                lockfile.set(libraryLockfile)
            }
        tasks.register<Exec>("find${variant.name.replaceFirstChar(Char::titlecase)}LibraryVulnerabilities") {
            description = "Finds known vulnerabilities in the libraries of the APK with OSV-Scanner."
            group = "verification"
            dependsOn(writeLibraryLockfile)
            commandLine(
                rootProject.file("../../.toolchain/bin/osv-scanner").path,
                "scan",
                "source",
                "--lockfile",
                libraryLockfile.get().asFile.path,
            )
        }
    }
}

tasks.matching { it.name.contains("ArtProfile") }.configureEach { enabled = false }

tasks.withType<KotlinCompile>().configureEach {
    compilerOptions {
        jvmTarget.set(JvmTarget.JVM_17)
        allWarningsAsErrors.set(true)
    }
}
