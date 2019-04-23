pipeline {
    parameters {
        string(name: 'TEST_DOCKER_IP', defaultValue: '10.3.199.246', description: 'Docker setup IP address to test on', trim: true)
        string(name: 'TEST_NS_SINGLE', defaultValue: 'https://10.3.199.247:8443', description: 'Single NS API address', trim: true)
    }
    options {
        disableConcurrentBuilds()
    }
    agent {
        node {
            label 'solutions-126'
        }
    }
    stages {
        stage('Build development') {
            steps {
                sh 'make build-development'
            }
        }
        stage('Push [local registry]') {
            steps {
                sh 'make push-development'
            }
        }
        stage('Tests [unit]') {
            steps {
                sh 'make test-unit-container'
            }
        }
        stage('Tests [e2e-docker]') {
            steps {
                sh '''
                    cat >./tests/e2e/_configs/single-ns.yaml <<EOL
restIp: ${TEST_NS_SINGLE}             # [required] NexentaStor REST API endpoint(s)
username: admin                       # [required] NexentaStor REST API username
password: Nexenta@1                   # [required] NexentaStor REST API password
defaultDataset: testPool/testDataset  # [required] 'pool/dataset' to use
defaultDataIp: ${TEST_NS_SINGLE:8:-5} # [required] NexentaStor data IP or HA VIP
EOL
                    echo "Generated config file for tests:";
                    cat ./tests/e2e/_configs/single-ns.yaml;
                    TEST_DOCKER_IP=${TEST_DOCKER_IP} make test-e2e-docker-development-container;
                '''
            }
        }
        stage('Build production') {
            steps {
                sh 'make build-production'
            }
        }
        stage('Push [hub.docker.com]') {
            when {
                branch 'master'
            }
            environment {
                DOCKER = credentials('docker-hub-credentials')
            }
            steps {
                sh '''
                    docker login -u ${DOCKER_USR} -p ${DOCKER_PSW};
                    make push-production;
                '''
            }
        }
    }
}
