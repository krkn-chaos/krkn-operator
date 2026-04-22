#!/usr/bin/env python3
"""
krkn-operator-data-provider
gRPC server that provides data from Kubernetes clusters using krkn-lib
"""

import base64
import logging
import os
import subprocess
import tempfile
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

    def ExecuteKubectl(self, request, context):
        """
        Execute kubectl or oc command with read-only permissions

        Args:
            request: ExecuteKubectlRequest containing command details and kubeconfig
            context: gRPC context

        Returns:
            ExecuteKubectlResponse containing stdout/stderr in base64 and exit code
        """
        kubeconfig_file = None
        try:
            logger.info(f"Received ExecuteKubectl request: {request.command} {request.subcommand}")

            # Decode base64 kubeconfig
            kubeconfig_decoded = base64.b64decode(request.kubeconfig_base64).decode('utf-8')
            logger.debug("Kubeconfig decoded successfully")

            # Create temporary kubeconfig file
            kubeconfig_file = tempfile.NamedTemporaryFile(mode='w', delete=False, suffix='.kubeconfig')
            kubeconfig_file.write(kubeconfig_decoded)
            kubeconfig_file.close()
            logger.debug(f"Temporary kubeconfig created at {kubeconfig_file.name}")

            # Build command
            cmd = [request.command, request.subcommand]
            cmd.extend(request.args)

            # Add named flags (--flag value)
            for key, value in request.flags.items():
                cmd.append(f"--{key}")
                cmd.append(value)

            # Add boolean flags (--flag)
            for flag in request.boolean_flags:
                cmd.append(f"--{flag}")

            # Add kubeconfig flag
            cmd.append(f"--kubeconfig={kubeconfig_file.name}")

            logger.info(f"Executing command: {' '.join(cmd)}")

            # Set timeout (default 120 seconds)
            timeout_seconds = request.timeout_seconds if request.timeout_seconds > 0 else 120

            # Execute command
            result = subprocess.run(
                cmd,
                capture_output=True,
                timeout=timeout_seconds
            )

            logger.info(f"Command completed with exit code {result.returncode}")

            # Encode stdout and stderr to base64
            stdout_base64 = base64.b64encode(result.stdout).decode('utf-8')
            stderr_base64 = base64.b64encode(result.stderr).decode('utf-8')

            # Return response
            response = dataprovider_pb2.ExecuteKubectlResponse(
                stdout_base64=stdout_base64,
                stderr_base64=stderr_base64,
                exit_code=result.returncode,
                error=""
            )
            return response

        except subprocess.TimeoutExpired:
            logger.error(f"Command execution timed out after {timeout_seconds}s")
            context.set_code(grpc.StatusCode.DEADLINE_EXCEEDED)
            context.set_details("Command execution timed out")
            return dataprovider_pb2.ExecuteKubectlResponse(
                stdout_base64="",
                stderr_base64="",
                exit_code=-1,
                error="timeout"
            )

        except FileNotFoundError:
            logger.error(f"Command not found: {request.command}")
            context.set_code(grpc.StatusCode.NOT_FOUND)
            context.set_details(f"Command '{request.command}' not found")
            return dataprovider_pb2.ExecuteKubectlResponse(
                stdout_base64="",
                stderr_base64="",
                exit_code=-1,
                error="not_found"
            )

        except Exception as e:
            logger.error(f"Error in ExecuteKubectl: {str(e)}", exc_info=True)
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Failed to execute command: {str(e)}")
            return dataprovider_pb2.ExecuteKubectlResponse(
                stdout_base64="",
                stderr_base64="",
                exit_code=-1,
                error="execution_error"
            )

        finally:
            # Clean up temporary kubeconfig file
            if kubeconfig_file and os.path.exists(kubeconfig_file.name):
                try:
                    os.unlink(kubeconfig_file.name)
                    logger.debug(f"Removed temporary kubeconfig {kubeconfig_file.name}")
                except Exception as e:
                    logger.warning(f"Failed to remove temporary kubeconfig: {str(e)}")


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