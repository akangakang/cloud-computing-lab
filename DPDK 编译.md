1. 增加`~/Desktop/cloud-computing-lab/dpdk/examples`路径`meson.build`文件中的all_examples

2. 在`~/Desktop/cloud-computing-lab/dpdk/build`路径下

   ```bash
    sudo meson configure -Dexamples=all
   ```

3. 在`~/Desktop/cloud-computing-lab/dpdk/build`路径下

   ```bash
   sudo ninja
   ```

4. ```bash
   cd example
   sudo ./dpdk-文件名
   ```

   