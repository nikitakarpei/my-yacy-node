plugins {
    id("companion.android-application")
}

android {
    namespace = "io.github.nikitakarpei.yacypeercompanion"

    defaultConfig {
        applicationId = "io.github.nikitakarpei.yacypeercompanion"
        versionCode = 1
        versionName = "0.1.0"
    }
}

dependencies {
    implementation(project(":services"))
    implementation(project(":attestedstatements"))
    implementation("com.google.zxing:core:3.5.4")
}
