import com.azure.resourcemanager.AzureResourceManager
import org.springframework.beans.factory.annotation.Autowired
import org.springframework.boot.test.context.SpringBootTest
import spock.lang.Specification
import uk.edoatley.azure.it.AzureResourceManagerConfiguration
import uk.edoatley.azure.it.AzureConfigurationProperties
import org.springframework.beans.factory.annotation.Autowired
import org.springframework.boot.context.properties.EnableConfigurationProperties

@SpringBootTest(classes = AzureResourceManagerConfiguration.class)
@EnableConfigurationProperties(AzureConfigurationProperties.class)
class AzureIntegrationTest extends Specification {

    @Autowired
    private AzureResourceManager azureResourceManager

    @Autowired
    private AzureConfigurationProperties config

    def "Resource group exists"() {
        when:
        def groups = azureResourceManager.resourceGroups()

        then:
        groups.list().size() > 0
        groups.list().any { it.name() == config.resourceGroupName }

    }

     def "VNET has correct name and IP ranges"() {

        when:
        def vnet = azureResourceManager.networks().getByResourceGroup(config.resourceGroupName, config.vnetName)

        then:
        vnet.name() == config.vnetName
        vnet.addressSpaces().size() == 1
        vnet.addressSpaces().any { it == config.vnetAddressSpace}
    }   

    def "Subnets have the correct names and IP ranges"() {
        when:
        def vnet = azureResourceManager.networks().getByResourceGroup(config.resourceGroupName, config.vnetName)
        def subnets = vnet.subnets()

        then:
        subnets.size() == 2
    
        subnets.each { name, subnet ->
            subnet.addressPrefix() == config.subnets[name]
        }
    }

    def "Check important IP address is free"() {
        
        when:
        def vnet = azureResourceManager.networks().getByResourceGroup(config.resourceGroupName, config.vnetName)

        then:
        vnet.isPrivateIPAddressAvailable(config.importantIpAddress)
    }
}
