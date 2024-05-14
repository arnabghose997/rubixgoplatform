# This script packs all the 

import os
import zipfile

def collect_and_zip_logs(base_dir, zip_name):
    # Create a ZipFile object in write mode
    with zipfile.ZipFile(zip_name, 'w') as zipf:
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
                            # Rename the log file for the zip archive
                            new_log_name = f'log_{item}.txt'
                            # Write the renamed log file to the zip archive
                            zipf.write(log_file_path, new_log_name)
                            print(f'Added {new_log_name} to {zip_name}')
                        else:
                            print(f'log.txt not found in {item_path}')

if __name__ == "__main__":
    base_directory = '..'  # Set to parent directory of 'tests' where env directories are located
    zip_filename = 'collected_logs.zip'
    collect_and_zip_logs(base_directory, zip_filename)
    print(f'All logs collected and zipped into {zip_filename}')