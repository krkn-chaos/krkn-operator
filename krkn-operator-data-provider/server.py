#!/usr/bin/env python3
"""
krkn-operator-data-provider
gRPC server that provides data from Kubernetes clusters using krkn-lib
"""

import base64
import logging
from concurrent import futures

import grpc
from generated import dataprovider_pb2, dataprovider_pb2_grpc
from krkn_lib.k8s import KrknKubernetes


# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class DataProviderServicer(dataprovider_pb2_grpc.DataProviderServiceServicer):
    """Implementation of DataProviderService"""

    def GetNodes(self, request, context):
        """
        Get list of nodes from a Kubernetes cluster

        Args:
            request: GetNodesRequest containing kubeconfig in base64
            context: gRPC context

        Returns:
            GetNodesResponse containing list of node names
        """
        try:
            logger.info("Received GetNodes request")

            # Decode base64 kubeconfig
            kubeconfig_decoded = base64.b64decode(request.kubeconfig_base64).decode('utf-8')
            logger.debug("Kubeconfig decoded successfully")

            # Initialize KrknKubernetes with the kubeconfig string
            krkn_k8s = KrknKubernetes(kubeconfig_path="",kubeconfig_string=kubeconfig_decoded)
            logger.info("KrknKubernetes initialized successfully")
            logger.info(f"kubeconfig {kubeconfig_decoded}")

            # Get list of nodes
            nodes = krkn_k8s.list_nodes()
            logger.info(f"Retrieved {len(nodes)} nodes from cluster")

            # Return response
            response = dataprovider_pb2.GetNodesResponse(nodes=nodes)
            return response

        except Exception as e:
            logger.error(f"Error in GetNodes: {str(e)}", exc_info=True)
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Failed to get nodes: {str(e)}")
            return dataprovider_pb2.GetNodesResponse()


def serve(port=50051):
    """
    Start the gRPC server

    Args:
        port: Port to listen on (default: 50051)
    """
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    dataprovider_pb2_grpc.add_DataProviderServiceServicer_to_server(
        DataProviderServicer(), server
    )

    server_address = f'[::]:{port}'
    server.add_insecure_port(server_address)

    logger.info(f"Starting gRPC server on {server_address}")
    server.start()
    logger.info("gRPC server started successfully")

    try:
        server.wait_for_termination()
    except KeyboardInterrupt:
        logger.info("Shutting down gRPC server...")
        server.stop(0)
        logger.info("gRPC server stopped")


if __name__ == '__main__':
    serve()