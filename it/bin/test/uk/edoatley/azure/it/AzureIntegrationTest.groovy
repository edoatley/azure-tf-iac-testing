
import com.azure.resourcemanager.AzureResourceManager
import org.springframework.beans.factory.annotation.Autowired
import org.springframework.boot.test.context.SpringBootTest
import spock.lang.Specification
import uk.edoatley.azure.it.AzureResourceManagerConfiguration

@SpringBootTest(classes = AzureResourceManagerConfiguration.class)
class AzureIntegrationTest extends Specification {

    @Autowired
    private AzureResourceManager azureResourceManager

    def "Resource group exists"() {
        when:
        def groups = azureResourceManager.resourceGroups()

        then:
        groups.list().size() > 0
        groups.list().any { it.name() == "rg-edo-dev-testapp"}

    }
}
