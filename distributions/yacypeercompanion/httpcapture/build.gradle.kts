plugins {
    id("companion.android-library")
}

android {
    namespace = "io.github.nikitakarpei.httpcapture"
}

dependencies {
    api(project(":attestedstatements"))
}
