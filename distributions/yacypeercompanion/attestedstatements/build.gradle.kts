plugins {
    id("companion.android-library")
}

android {
    namespace = "io.github.nikitakarpei.attestedstatements"
}

dependencies {
    api("com.squareup.okhttp3:okhttp:5.5.1")
    implementation("org.bouncycastle:bcpkix-jdk18on:1.86")
}
