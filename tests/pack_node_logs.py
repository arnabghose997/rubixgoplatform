# This script packs all the 

import os
import shutil

def collect_logs(base_dir, output_dir):
    # Ensure the output directory exists
    os.makedirs(output_dir, exist_ok=True)
    
    # Iterate over all items in the base directory
    for env in ['linux', 'windows', 'mac']:
        env_path = os.path.join(base_dir, env)
        if os.path.isdir(env_path):
            for item in os.listdir(env_path):
                item_path = os.path.join(env_path, item)
                # Check if the item is a directory and its name starts with 'node'
                if os.path.isdir(item_path) and item.startswith('node'):
                    log_file_path = os.path.join(item_path, 'log.txt')
                    if os.path.exists(log_file_path):
                        # Rename the log file
                        new_log_name = f'log_{env}_{item}.txt'
                        new_log_path = os.path.join(output_dir, new_log_name)
                        # Copy the log file to the output directory
                        shutil.copyfile(log_file_path, new_log_path)
                        print(f'Copied {new_log_name} to {output_dir}')
                    else:
                        print(f'log.txt not found in {item_path}')

if __name__ == "__main__":
    base_directory = '..'  # Parent directory of 'tests' where env directories are located
    output_directory = './node_logs'  # Directory to store collected logs
    collect_logs(base_directory, output_directory)
    print(f'All logs collected into {output_directory}')