const cleanCommand = (): void => {
  const isDeleteAllLogFiles = confirm('Do you want to erase all the files')

  if (isDeleteAllLogFiles === true) {
    // TODO delete
    console.log('deleted all files')
    Deno.exit(0)
  } else {
    console.log('cancelled')
    Deno.exit(0)
  }
}

cleanCommand()