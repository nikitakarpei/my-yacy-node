package io.github.nikitakarpei.pagescrape

import io.github.nikitakarpei.servicehost.ServiceApp
import io.github.nikitakarpei.servicehost.ServiceHost
import io.github.nikitakarpei.servicehost.ServiceInstance
import io.github.nikitakarpei.servicehost.ServiceScreen
import pagescrape.Pagescrape

object PagescrapeApp : ServiceApp("pagescrape") {
    override val hostClass = PagescrapeHost::class.java

    override val screenClass = MainActivity::class.java

    override fun variables(): String = Pagescrape.variables()

    override fun instanceFrom(environment: String): ServiceInstance {
        val instance = Pagescrape.instanceFrom(environment)
        return object : ServiceInstance {
            override fun run() = instance.run()

            override fun stop() = instance.stop()
        }
    }
}

class PagescrapeHost : ServiceHost() {
    override val app = PagescrapeApp
}

class MainActivity : ServiceScreen() {
    override val app = PagescrapeApp
}
