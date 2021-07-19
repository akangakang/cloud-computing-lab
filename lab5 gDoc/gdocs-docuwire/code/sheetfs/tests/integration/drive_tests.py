import contextlib
import dataclasses
import logging
import os
import shlex
import subprocess as sp
import time
from typing import TypeVar, Generic

import pexpect
from kazoo.client import KazooClient
from kazoo.exceptions import NoNodeError

logging.basicConfig(format='%(asctime)s:%(levelname)s:%(message)s', level=logging.DEBUG)


@contextlib.contextmanager
def cd(path: str):
    cwd = os.getcwd()
    os.chdir(path)
    yield
    os.chdir(cwd)


@dataclasses.dataclass
class MasterConfig:
    addr_for_client: str
    datanode_groups: str
    election_ack: str
    election_znode: str
    node_id: str
    kafka_server: str
    kafka_topic: str
    port: int
    zookeeper_servers: str


@dataclasses.dataclass
class DataNodeConfig:
    addr_for_client: str
    port: int
    node_id: str
    group_name: str
    zookeeper_servers: str
    kafka_server: str


def _run_cmd_check_status(cmd: str) -> str:
    logging.debug(f'running cmd with return status: {cmd}')
    process = sp.run(cmd, shell=True, text=True)
    if process.returncode != 0:
        print(process.stdout)
        raise RuntimeError(f'Failed to run command {cmd}!')

    return process.stdout


def _build_master():
    with cd('../../master'):
        _run_cmd_check_status("go build")


def _build_datanode():
    with cd('../../datanode'):
        _run_cmd_check_status("go build")


def run_zk_kafka():
    _run_cmd_check_status("docker-compose down")
    _run_cmd_check_status("docker-compose up -d")


def _run_master(config: MasterConfig) -> sp.Popen:
    cmd = f'../../master/master -a "{config.addr_for_client}" -dngroups "{config.datanode_groups}" -elack "{config.election_ack}" -elznode "{config.election_znode}" -i "{config.node_id}" -kfserver "{config.kafka_server}" -kftopic "{config.kafka_topic}" -p {config.port} -zkservers "{config.zookeeper_servers}"'
    logging.debug(f'booting master using: {cmd}')
    return sp.Popen(shlex.split(cmd), text=True)


def _run_datanode(config: DataNodeConfig) -> sp.Popen:
    cmd = f'../../datanode/datanode -a "{config.addr_for_client}" -p "{config.port}" -i "{config.node_id}" -gn "{config.group_name}" -sl "{config.zookeeper_servers}" -ks "{config.kafka_server}"'
    logging.debug(f'booting datanode using: {cmd}')
    return sp.Popen(shlex.split(cmd), text=True)


T = TypeVar('T')


def check_primary(zk: KazooClient, ack: str, node_set: dict[str, T], retry: int = 10) -> tuple[T, list[T]]:
    for _ in range(retry):
        if zk.exists(ack):
            try:
                addr, _ = zk.get(ack)
            except NoNodeError:
                continue
            addr = addr.decode('utf-8')
            if addr in node_set:
                logging.debug(f'select {addr} as primary.')
                return node_set[addr], [v for k, v in node_set.items() if k != addr]
        time.sleep(1)
    raise RuntimeError("not found primary!")


@dataclasses.dataclass
class NodeSetItem(Generic[T]):
    process: sp.Popen
    cfg: T

    @property
    def caddr(self) -> str:
        return self.cfg.addr_for_client


MASTER_ELECTION_ZNODE = "/master-election"
MASTER_ELECTION_ACK = "/master-election-ack"
DATANODE_GROUP_ACK_PREFIX = "/datanode_election_ack_"
KAFKA_SERVER = "127.0.0.1:9093"
ZOOKEEPER_SERVER = "127.0.0.1:2181,127.0.0.1:2182,127.0.0.1:2183"
DATANODE_GROUPS = ["group1", "group2", "group3"]
DATA_DIRECTORY = "../data"

MasterSetT = dict[str, NodeSetItem[MasterConfig]]
DatanodeGroupSetT = dict[str, dict[str, NodeSetItem[DataNodeConfig]]]


def _run_masters(count: int, start_port: int) -> MasterSetT:
    res = {}
    for i in range(count):
        port = start_port + i
        caddr = f'127.0.0.1:{port}'
        cfg = MasterConfig(
            addr_for_client=caddr,
            datanode_groups=','.join(DATANODE_GROUPS),
            election_znode=MASTER_ELECTION_ZNODE,
            election_ack=MASTER_ELECTION_ACK,
            port=port,
            node_id=f'master{i}',
            zookeeper_servers=ZOOKEEPER_SERVER,
            kafka_server=KAFKA_SERVER,
            kafka_topic='master-journal'
        )
        res[caddr] = NodeSetItem(_run_master(cfg), cfg)
    return res


def _run_datanodes(count_per_group: int, start_port: int) -> dict[str, dict[str, NodeSetItem[DataNodeConfig]]]:
    res = {}
    for i, group in enumerate(DATANODE_GROUPS):
        group_node_set = {}
        for j in range(count_per_group):
            port = start_port + i * count_per_group + j
            caddr = f'127.0.0.1:{port}'
            cfg = DataNodeConfig(
                addr_for_client=caddr,
                port=port,
                node_id=f'datanode-{group}-{j}',
                zookeeper_servers=ZOOKEEPER_SERVER,
                kafka_server=KAFKA_SERVER,
                group_name=f'{group}'
            )
            group_node_set[caddr] = NodeSetItem(_run_datanode(cfg), cfg)
        res[group] = group_node_set
    return res


def _clean_test_data():
    sp.run('rm *.db', shell=True)
    sp.run(f'rm -rf {DATA_DIRECTORY}/*', shell=True)


class TestingEnvironment:
    master_set: MasterSetT
    datanode_group_set: DatanodeGroupSetT
    primary_master: NodeSetItem[MasterConfig]
    group_primary_datanode: dict[str, NodeSetItem[DataNodeConfig]]
    zk: KazooClient

    @staticmethod
    def _node_set_output(node_set: dict[str, NodeSetItem]):
        for v in node_set.values():
            logging.debug(f'{v.caddr}: {v.process.before}')

    def __init__(self):
        _clean_test_data()
        run_zk_kafka()
        logging.debug('waiting 5 secs for zookeepers and kafka preparation.')
        time.sleep(5)
        _build_master()
        self.master_set = _run_masters(3, 10000)
        _build_datanode()
        self.datanode_group_set = _run_datanodes(3, 15000)
        self.zk = KazooClient(hosts=ZOOKEEPER_SERVER)
        self.zk.start()
        self.primary_master, _ = check_primary(self.zk, MASTER_ELECTION_ACK, self.master_set)
        self.group_primary_datanode = {}
        for group in DATANODE_GROUPS:
            p, _ = check_primary(self.zk, f'{DATANODE_GROUP_ACK_PREFIX}{group}', self.datanode_group_set[group])
            self.group_primary_datanode[group] = p

    def crash_primary_master(self):
        self.primary_master.process.kill()
        del self.master_set[self.primary_master.caddr]
        self.primary_master, _ = check_primary(self.zk, MASTER_ELECTION_ACK, self.master_set)

    def crash_all_datanode_groups(self):
        for group in DATANODE_GROUPS:
            group_primary = self.group_primary_datanode[group]
            group_primary.process.kill()
            group_set = self.datanode_group_set[group]
            del group_set[group_primary.caddr]
            self.group_primary_datanode[group], _ = check_primary(self.zk, f'{DATANODE_GROUP_ACK_PREFIX}{group}',
                                                                  group_set)

    def destroy_nodes(self):
        for m in self.master_set.values():
            m.process.kill()

        for group in self.datanode_group_set.values():
            for n in group.values():
                n.process.kill()

    def output(self):
        self._node_set_output(self.master_set)

        for group in self.datanode_group_set.values():
            self._node_set_output(group)

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        self.destroy_nodes()
        self.zk.stop()
        return False


def direct_test(name: str):
    with TestingEnvironment() as t:
        try:
            out = _run_cmd_check_status(f"go test -run {name}")
        except RuntimeError:
            logging.error(f'\u001b[31m test {name} failed: {out} \u001b[0m')
        else:
            logging.info(f'\u001b[32m test {name} passed. \u001b[0m')


def test_handle_crash():
    with TestingEnvironment() as t:
        test_process = pexpect.spawn("go test -run TestHandleCrash", timeout=30)

        test_process.expect('Please crash datanodes!')

        logging.debug("instruction received, start crashing all datanodes")
        time.sleep(5)

        t.crash_all_datanode_groups()
        logging.debug('Crashed all datanode groups.')

        test_process.sendline()

        test_process.expect("Please crash datanodes and master!")

        logging.debug("instruction received, start crashing all datanodes")
        time.sleep(5)

        t.crash_all_datanode_groups()
        t.crash_primary_master()
        logging.debug('Crashed all datanode groups and current primary master.')

        test_process.sendline()
        try:
            test_process.expect('PASS')
        except pexpect.TIMEOUT:
            logging.error('\u001b[31m test TestHandleCrash timeout. \u001b[0m')
            test_process.close(True)
        logging.debug(test_process.read())
        if test_process.status != 0:
            logging.error('\u001b[31m test TestHandleCrash failed. \u001b[0m')
        else:
            logging.info(f'\u001b[32m test TestHandleCrash passed. \u001b[0m')


def main():
    direct_test('TestCreate')
    direct_test('TestOpen')
    direct_test('TestReadAndWrite')
    direct_test('TestComplicatedReadAndWrite')
    direct_test('TestConcurrentWrite')
    test_handle_crash()


if __name__ == '__main__':
    main()
