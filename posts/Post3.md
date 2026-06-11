# Validating Azure Resources with Spock

- [Validating Azure Resources with Spock](#validating-azure-resources-with-spock)
  - [Introduction](#introduction)
  - [Steps](#steps)
    - [Pre-requisites](#pre-requisites)
    - [Step 1 - Create a new directory and initialise gradle](#step-1---create-a-new-directory-and-initialise-gradle)
    - [Step 2. Configure the gradle build](#step-2-configure-the-gradle-build)
    - [Step 3. Create a basic test](#step-3-create-a-basic-test)
    - [Step 4. Try the test out to check it fails as expected](#step-4-try-the-test-out-to-check-it-fails-as-expected)
    - [Step 5. Fix the test](#step-5-fix-the-test)
    - [Step 6. Creating more configurable tests](#step-6-creating-more-configurable-tests)
    - [Step 7. Creating more realistic / useful tests](#step-7-creating-more-realistic--useful-tests)
  - [Conclusion](#conclusion)


## Introduction

In my [previous article](https://www.linkedin.com/pulse/testing-iac-using-terratest-ed-oatley-4arje) I explained how I had used Terragrunt to build Azure resources
and then used terratest to check that the terraform code is doing what I'd expected it to and define the correct infrastructure.

One challenge with the terratest approach is that it cannot tell you if what is deployed meets you expectations. Some
scenarios that may resonate are:
a) somebody manually made a change in the portal 
b) you want to check a small change has been applied without building the entire infrastructure from scratch with terraform 
c) you want to check a resource property that is not easy to get hold of from the terraform output or is not currently available
d) you only see a problem in a certain environment

In these cases directly observing the real resources rather than those created by a test run with terratest. The approach that we will use here will avoid this 
shortcoming by defining tests using the [Spock](http://spockframework.org/)  framework interacting with Azure (ARM) via the Java SDK. In this way, we can directly
query the resources and assert that they fulfil our expectations.

## Steps

### Pre-requisites

We will start with the same basic terragrunt set up I used in the last two articles where we have a `dev` and a `prod` environment each to contain a resource group, a virtual network and a virtual machine:

```bash
❯ cd terraform/
❯ tree .
.
├── environments
│   ├── dev
│   │   ├── dev-common.yaml
│   │   ├── resource_group
│   │   │   └── terragrunt.hcl
│   │   ├── terragrunt.hcl
│   │   ├── virtual_machine
│   │   │   └── terragrunt.hcl
│   │   └── virtual_network
│   │       └── terragrunt.hcl
│   ├── prod
│   │   ├── prod-common.yaml
│   │   ├── resource_group
│   │   │   └── terragrunt.hcl
│   │   ├── terragrunt.hcl
│   │   ├── virtual_machine
│   │   │   └── terragrunt.hcl
│   │   └── virtual_network
│   │       └── terragrunt.hcl
│   └── terragrunt.hcl
└── modules
    ├── resource_group
    │   ├── main.tf
    │   ├── outputs.tf
    │   └── variables.tf
    ├── vm
    │   ├── main.tf
    │   ├── outputs.tf
    │   └── variables.tf
    └── vnet
        ├── main.tf
        ├── outputs.tf
        └── variables.tf
```

### Step 1 - Create a new directory and initialise gradle

Here we initialise our gradle project:

```bash
> mkdir it && cd it
> gradle init
Starting a Gradle Daemon (subsequent builds will be faster)

Select type of project to generate:
  1: basic
  2: application
  3: library
  4: Gradle plugin
Enter selection (default: basic) [1..4] 1

Select build script DSL:
  1: Groovy
  2: Kotlin
Enter selection (default: Groovy) [1..2] 1

Project name (default: it): 

> Task :init
Get more help with your project: Learn more about Gradle by exploring our samples at https://docs.gradle.org/6.7/samples

BUILD SUCCESSFUL in 17s
2 actionable tasks: 2 executed
```

### Step 2. Configure the gradle build

Firstly we shall update the `gradle` version by running:

```bash
./gradlew wrapper --gradle-version 8.4
```

Now we can populate the generated `build.gradle` file:

```groovy
plugins {
    id 'java'
    id 'groovy'
    id("com.adarshr.test-logger") version "3.2.0"
    id("io.freefair.lombok") version "8.3"
}

group = 'uk.edoatley'
version = '0.0.1-SNAPSHOT'
sourceCompatibility = JavaVersion.VERSION_17

repositories {
    mavenCentral()
    jcenter()
}

dependencies {
    // Azure deps
    implementation 'com.azure:azure-identity:1.8.2'
    implementation 'com.azure:azure-core-http-netty:1.13.1'
    implementation 'com.azure.resourcemanager:azure-resourcemanager:2.30.0'
    implementation 'com.azure.resourcemanager:azure-resourcemanager-security:1.0.0-beta.5'


    // Spock
    testImplementation 'org.codehaus.groovy:groovy-all:3.0.16'
    testImplementation 'org.spockframework:spock-core:2.3-groovy-3.0'

    // add SpringBoot support for ease of configurability
    testImplementation 'org.springframework.boot:spring-boot-starter-test:2.7.15'
    testImplementation 'org.spockframework:spock-spring:2.3-groovy-3.0'
}


test {
    useJUnitPlatform()
    testLogging { 
        showStandardStreams = true
        exceptionFormat "full"
        minGranularity = 3
    }
}
```

and we can validate that is all good with:

```bash
./gradlew clean build
```

### Step 3. Create a basic test

Firstly let's create our directory structure:

```bash
mkdir -p src/test/groovy/uk/edoatley/azure/it
mkdir -p src/test/java/uk/edoatley/azure/it
mkdir -p src/test/resources
```

Next we create our configuration class in java as personally I found it simpler:

```java
@Configuration
public class AzureResourceManagerConfiguration {

    @Value("${azure.tenant}")
    private String tenant;

    @Value("${azure.subscription}")
    private String subscription;

    @Bean
    public AzureResourceManager azureResourceManager() {
        AzureProfile toolingAzureProfile = new AzureProfile(tenant, subscription, AzureEnvironment.AZURE);
        DefaultAzureCredential toolingCredential = new DefaultAzureCredentialBuilder()
                .authorityHost(toolingAzureProfile.getEnvironment().getActiveDirectoryEndpoint())
                .build();
        return AzureResourceManager
                .authenticate(toolingCredential, toolingAzureProfile)
                .withDefaultSubscription();
    }
}
```

This class is creating an [AzureResourceManager](https://learn.microsoft.com/en-us/java/api/overview/azure/resourcemanager-readme?view=azure-java-stable) bean 
that we can use to query ARM for details of the deployed infrastructure. We can then create our spock Specification i.e. test:

```groovy
@SpringBootTest
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
```

finally we need to create an `application.yaml` file like this in `src/test/resources` setting the tenant and subscription values:

```yaml
azure:
  tenant: "11111111-1111-1111-1111-111111111111"
  subscription: "22222222-2222-2222-2222-222222222222"
```

### Step 4. Try the test out to check it fails as expected

We can now try to run our test and check it fails (as we have not deployed anything yet!)

```bash
> ./gradlew clean build
...
AzureIntegrationTest > Resource group exists STANDARD_OUT
    2023-10-19 17:06:23.463  INFO 378627 --- [    Test worker] c.azure.identity.ChainedTokenCredential  : Azure Identity => Attempted credential EnvironmentCredential is unavailable.
    2023-10-19 17:06:24.536  INFO 378627 --- [onPool-worker-1] c.azure.identity.ChainedTokenCredential  : Azure Identity => Attempted credential ManagedIdentityCredential is unavailable.
    2023-10-19 17:06:24.558  INFO 378627 --- [onPool-worker-1] c.azure.identity.ChainedTokenCredential  : Azure Identity => Attempted credential SharedTokenCacheCredential is unavailable.
    2023-10-19 17:06:24.587  INFO 378627 --- [onPool-worker-1] c.azure.identity.ChainedTokenCredential  : Azure Identity => Attempted credential IntelliJCredential is unavailable.
    2023-10-19 17:06:25.039  INFO 378627 --- [onPool-worker-1] com.azure.identity.AzureCliCredential    : Azure Identity => getToken() result for scopes [https://management.core.windows.net//.default]: SUCCESS
    2023-10-19 17:06:25.039  INFO 378627 --- [onPool-worker-1] c.azure.identity.ChainedTokenCredential  : Azure Identity => Attempted credential AzureCliCredential returns a token
    2023-10-19 17:06:25.040  INFO 378627 --- [onPool-worker-1] c.a.c.implementation.AccessTokenCache    : {"az.sdk.message":"Acquired a new access token."}
AzureIntegrationTest

  Test Resource group exists FAILED (2.3s)

  Condition not satisfied:

  groups.list().any { it.name() == "rg-edo-dev-testapp"}
  |      |      |
  |      |      false
  |      <com.azure.resourcemanager.resources.fluentcore.utils.PagedConverter$PagedIterableImpl@52c2d1e8 pagedIterable=com.azure.core.http.rest.PagedIterable@1a67bcaf mapper=com.azure.resourcemanager.resources.fluentcore.arm.collection.implementation.ReadableWrappersImpl$$Lambda$725/0x0000000801758678@4f4b7a2e pageMapper=com.azure.resourcemanager.resources.fluentcore.utils.PagedConverter$PagedIterableImpl$$Lambda$727/0x0000000801758f98@334ac669 pagedFlux=PagedFlux firstPageRetriever=null nextPageRetriever=null pagedFlux=PagedFlux batchSize=1 pageRetrieverSyncProvider=null defaultPageSize=null continuationPredicate=null flux=PagedFlux iterable=null>
  <com.azure.resourcemanager.resources.implementation.ResourceGroupsImpl@2740585b logger=com.azure.core.util.logging.ClientLogger@5aa461 resourceManager=com.azure.resourcemanager.resources.ResourceManager@f3e6876>
      at AzureIntegrationTest.Resource group exists(AzureIntegrationTest.groovy:20)

FAILURE: Executed 1 tests in 5.8s (1 failed)
```

We can see here there is no resource group named "rg-edo-dev-testapp" is the subscription we configured. 

One other thing to note here in the output is the mention of [`DefaultAzureCredential`](https://docs.microsoft.com/en-us/java/api/com.azure.identity.defaultazurecredential?view=azure-java-stable) 
which is a very useful class that will try a number of different ways to authenticate with Azure. In this case it is using the Azure CLI login but in a CI build it could use a manged identity.

### Step 5. Fix the test

Firstly lets create the infrastructure we want to test by running the apply:

```bash
> cd terraform/environments/dev
> terragrunt run-all apply
...
Apply complete! Resources: 5 added, 0 changed, 0 destroyed.

Outputs:

subnet_address_spaces = {
  "subnet1" = tolist([
    "10.0.1.0/24",
  ])
  "subnet2" = tolist([
    "10.0.2.0/24",
  ])
}
subnet_ids = {
  "subnet1" = "/subscriptions/22222222-2222-2222-2222-222222222222/resourceGroups/rg-edo-dev-testapp/providers/Microsoft.Network/virtualNetworks/vnet-edo-dev-testapp/subnets/subnet1"
  "subnet2" = "/subscriptions/22222222-2222-2222-2222-222222222222/resourceGroups/rg-edo-dev-testapp/providers/Microsoft.Network/virtualNetworks/vnet-edo-dev-testapp/subnets/subnet2"
}
vnet_address_space = tolist([
  "10.0.0.0/16",
])
vnet_id = "/subscriptions/22222222-2222-2222-2222-222222222222/resourceGroups/rg-edo-dev-testapp/providers/Microsoft.Network/virtualNetworks/vnet-edo-dev-testapp"
vnet_name = "vnet-edo-dev-testapp"
```

Now we can run our test again and see it pass:

```bash
> ./gradlew clean build
...
AzureIntegrationTest > Resource group exists STANDARD_OUT
    2023-10-19 17:52:32.346  INFO 391518 --- [    Test worker] c.azure.identity.ChainedTokenCredential  : Azure Identity => Attempted credential EnvironmentCredential is unavailable.
    2023-10-19 17:52:33.557  INFO 391518 --- [onPool-worker-1] c.azure.identity.ChainedTokenCredential  : Azure Identity => Attempted credential ManagedIdentityCredential is unavailable.
    2023-10-19 17:52:33.582  INFO 391518 --- [onPool-worker-1] c.azure.identity.ChainedTokenCredential  : Azure Identity => Attempted credential SharedTokenCacheCredential is unavailable.
    2023-10-19 17:52:33.610  INFO 391518 --- [onPool-worker-1] c.azure.identity.ChainedTokenCredential  : Azure Identity => Attempted credential IntelliJCredential is unavailable.
    2023-10-19 17:52:34.121  INFO 391518 --- [onPool-worker-1] com.azure.identity.AzureCliCredential    : Azure Identity => getToken() result for scopes [https://management.core.windows.net//.default]: SUCCESS
    2023-10-19 17:52:34.122  INFO 391518 --- [onPool-worker-1] c.azure.identity.ChainedTokenCredential  : Azure Identity => Attempted credential AzureCliCredential returns a token
    2023-10-19 17:52:34.123  INFO 391518 --- [onPool-worker-1] c.a.c.implementation.AccessTokenCache    : {"az.sdk.message":"Acquired a new access token."}
AzureIntegrationTest

  Test Resource group exists PASSED (2.5s)

SUCCESS: Executed 1 tests in 7s
```

### Step 6. Creating more configurable tests

The test above is great and gives us confidence our infrastructure really deployed and was correctly configured. However, it is not very useful
across many environments as it is hard coded to the dev environment. The terragrunt definitions allows us to deploy to many environments and so
it would be good if our tests did the same. To do this we can utilise Spring properties as follows:

```java
package uk.edoatley.azure.it;
import java.util.Map;
import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.stereotype.Component;
import lombok.Data;

@Data
@Component
@ConfigurationProperties(prefix = "azure.config")
public class AzureConfigurationProperties {
    private String resourceGroupName;
    private String vnetName;
    private String vnetAddressSpace;
    private Map<String, String> subnets;
    private String importantIpAddress;
}
```

and corresponding yaml configuration for the `dev` environment in file `src/test/resources/application-dev.yaml` which will override override those in `application.yaml`:

```yaml
azure:
  config:
    resource-group-name: rg-edo-dev-testapp
    vnet-name: vnet-edo-dev-testapp
    vnet-address-space: "10.0.0.0/16"
    subnets:
      subnet1: "10.0.1.0/24"
      subnet2: "10.0.2.0/24"
    important-ip-address: "10.0.1.24"
```

We can now update our test to use the properties, for example:

```groovy
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
}
```

The key things here are to Autowire in the properties class and use the @EnableConfigurationProperties annotation to tell Spring to populate it.

We can now run the test with the dev profile and see it pass:

```bash
❯ SPRING_PROFILES_ACTIVE=dev ./gradlew clean build
...
AzureIntegrationTest

  Test Resource group exists PASSED (2.3s)

SUCCESS: Executed 1 tests in 6.6s
```

Extending our tests and running them accross many environments will now be greatly simplified.

### Step 7. Creating more realistic / useful tests

Now that we have a basic test working we can start to add more tests to validate the infrastructure we have deployed.

Let's start by replicating the ones we created with Terratest:

```groovy
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
```

## Conclusion

Spock and the ARM SDK can be used to define some very useful tests that help fill in any gaps left by terratest helping us to ensure that the 
Infrastructure-as-code we define is doing what we really want it to. The source code for all of the articles in this series can be found on 
GitHub here [edoatley/azure-tf-iac-testing](https://github.com/edoatley/azure-tf-iac-testing).