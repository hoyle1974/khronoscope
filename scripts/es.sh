# Create a directory on your host for Elasticsearch data (if it doesn't exist)
mkdir -p elasticsearch-data

docker stop elasticsearch && docker rm elasticsearch
docker run -d --name elasticsearch -p 9200:9200 -p 9300:9300 \
    -e "discovery.type=single-node" \
    -e "xpack.security.enabled=false" \
    -v `pwd`/elasticsearch-data:/usr/share/elasticsearch/data \
    docker.elastic.co/elasticsearch/elasticsearch:8.12.0
